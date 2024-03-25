package simulator

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/ledgerwatch/log/v3"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/ledgerwatch/erigon-lib/chain/snapcfg"
	"github.com/ledgerwatch/erigon-lib/common/background"
	"github.com/ledgerwatch/erigon-lib/downloader/snaptype"
	"github.com/ledgerwatch/erigon/eth/ethconfig"
	"github.com/ledgerwatch/erigon/polygon/heimdall"
	"github.com/ledgerwatch/erigon/turbo/snapshotsync/freezeblocks"
)

type HeimdallSimulator struct {
	ctx                context.Context
	knownSnapshots     *freezeblocks.RoSnapshots
	activeSnapshots    *freezeblocks.RoSnapshots
	knownBorSnapshots  *freezeblocks.BorRoSnapshots
	activeBorSnapshots *freezeblocks.BorRoSnapshots
	blockReader        *freezeblocks.BlockReader
	logger             log.Logger
	downloader         *TorrentClient

	nextSpan                  uint64
	lastDownloadedBlockNumber uint64
}

type IndexFnType func(context.Context, snaptype.FileInfo, string, *background.Progress, log.Lvl, log.Logger) error

func NewHeimdall(ctx context.Context, chain string, snapshotLocation string, logger log.Logger) (HeimdallSimulator, error) {
	cfg := snapcfg.KnownCfg(chain)

	knownSnapshots := freezeblocks.NewRoSnapshots(ethconfig.Defaults.Snapshot, snapshotLocation, 0, logger)
	knownBorSnapshots := freezeblocks.NewBorRoSnapshots(ethconfig.Defaults.Snapshot, snapshotLocation, 0, logger)

	files := make([]string, 0, len(cfg.Preverified))

	for _, item := range cfg.Preverified {
		files = append(files, item.Name)
	}

	knownSnapshots.InitSegments(files)
	knownBorSnapshots.InitSegments(files)

	activeSnapshots := freezeblocks.NewRoSnapshots(ethconfig.Defaults.Snapshot, snapshotLocation, 0, logger)
	activeBorSnapshots := freezeblocks.NewBorRoSnapshots(ethconfig.Defaults.Snapshot, snapshotLocation, 0, logger)

	if err := activeSnapshots.ReopenFolder(); err != nil {
		return HeimdallSimulator{}, err
	}
	if err := activeBorSnapshots.ReopenFolder(); err != nil {
		return HeimdallSimulator{}, err
	}

	downloader, err := NewTorrentClient(ctx, chain, snapshotLocation, logger)
	if err != nil {
		return HeimdallSimulator{}, err
	}

	s := HeimdallSimulator{
		ctx:                ctx,
		knownSnapshots:     knownSnapshots,
		activeSnapshots:    activeSnapshots,
		knownBorSnapshots:  knownBorSnapshots,
		activeBorSnapshots: activeBorSnapshots,
		blockReader:        freezeblocks.NewBlockReader(activeSnapshots, activeBorSnapshots),
		logger:             logger,
		downloader:         downloader,
		nextSpan:           0,
	}

	go func() {
		<-ctx.Done()
		s.Close()
	}()

	return s, nil
}

// FetchLatestSpan gets the next span from the snapshot
func (h *HeimdallSimulator) FetchLatestSpan(ctx context.Context) (*heimdall.Span, error) {
	span, err := h.getSpan(h.ctx, h.nextSpan)
	if err != nil {
		return nil, err
	}

	h.nextSpan++
	return &span, nil
}

func (h *HeimdallSimulator) FetchSpan(ctx context.Context, spanID uint64) (*heimdall.Span, error) {
	if spanID > h.nextSpan-1 {
		return nil, errors.New("span not found")
	}

	span, err := h.getSpan(h.ctx, spanID)
	if err != nil {
		return nil, err
	}

	return &span, err
}

func (h *HeimdallSimulator) FetchStateSyncEvents(ctx context.Context, fromId uint64, to time.Time, limit int) ([]*heimdall.EventRecordWithTime, error) {
	events, err := h.getEvents(h.ctx, fromId, to, limit)
	return events, err
}

func (h *HeimdallSimulator) FetchCheckpoint(ctx context.Context, number int64) (*heimdall.Checkpoint, error) {
	return nil, status.Errorf(codes.Unimplemented, "method FetchCheckpoint not implemented")
}

func (h *HeimdallSimulator) FetchCheckpointCount(ctx context.Context) (int64, error) {
	return 0, status.Errorf(codes.Unimplemented, "method FetchCheckpointCount not implemented")
}

func (h *HeimdallSimulator) FetchMilestone(ctx context.Context, number int64) (*heimdall.Milestone, error) {
	return nil, status.Errorf(codes.Unimplemented, "method FetchMilestone not implemented")
}

func (h *HeimdallSimulator) FetchMilestoneCount(ctx context.Context) (int64, error) {
	return 0, status.Errorf(codes.Unimplemented, "method FetchMilestoneCount not implemented")
}

func (h *HeimdallSimulator) FetchNoAckMilestone(ctx context.Context, milestoneID string) error {
	return status.Errorf(codes.Unimplemented, "method FetchNoAckMilestone not implemented")
}

func (h *HeimdallSimulator) FetchLastNoAckMilestone(ctx context.Context) (string, error) {
	return "", status.Errorf(codes.Unimplemented, "method FetchLastNoAckMilestone not implemented")
}

func (h *HeimdallSimulator) FetchMilestoneID(ctx context.Context, milestoneID string) error {
	return status.Errorf(codes.Unimplemented, "method FetchMilestoneID not implemented")
}

func (h *HeimdallSimulator) Close() {
	h.activeSnapshots.Close()
	h.knownSnapshots.Close()
}

func (h *HeimdallSimulator) downloadData(ctx context.Context, spans *freezeblocks.Segment, sType snaptype.Type, indexFn IndexFnType) error {
	fileName := snaptype.SegmentFileName(1, spans.From(), spans.To(), sType.Enum())

	h.logger.Warn(fmt.Sprintf("Downloading %s", fileName))

	err := h.downloader.Download(ctx, fileName)
	if err != nil {
		return fmt.Errorf("can't download %s: %w", fileName, err)
	}

	h.logger.Warn(fmt.Sprintf("Indexing %s", fileName))

	info, _, _ := snaptype.ParseFileName(h.downloader.LocalFsRoot(), fileName)

	err = indexFn(ctx, info, h.downloader.LocalFsRoot(), nil, log.LvlWarn, h.logger)
	if err != nil {
		return fmt.Errorf("can't download %s: %w", fileName, err)
	}

	return h.activeBorSnapshots.ReopenSegments([]snaptype.Type{sType}, true)
}
