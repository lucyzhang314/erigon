package snaptype

import (
	"bytes"
	"context"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"path/filepath"
	"runtime"
	"time"

	"github.com/ledgerwatch/erigon-lib/chain"
	"github.com/ledgerwatch/erigon-lib/chain/networkname"
	"github.com/ledgerwatch/erigon-lib/chain/snapcfg"
	"github.com/ledgerwatch/erigon-lib/common"
	"github.com/ledgerwatch/erigon-lib/common/background"
	"github.com/ledgerwatch/erigon-lib/common/dbg"
	"github.com/ledgerwatch/erigon-lib/common/hexutility"
	"github.com/ledgerwatch/erigon-lib/common/length"
	"github.com/ledgerwatch/erigon-lib/downloader/snaptype"
	"github.com/ledgerwatch/erigon-lib/kv"
	"github.com/ledgerwatch/erigon-lib/recsplit"
	"github.com/ledgerwatch/erigon-lib/seg"
	"github.com/ledgerwatch/erigon/core/rawdb"
	bor_types "github.com/ledgerwatch/erigon/polygon/bor/types"
	"github.com/ledgerwatch/erigon/polygon/heimdall"
	"github.com/ledgerwatch/log/v3"
)

func init() {
	borTypes := append(snaptype.BlockSnapshotTypes, BorSnapshotTypes...)

	snapcfg.RegisterKnownTypes(networkname.MumbaiChainName, borTypes)
	snapcfg.RegisterKnownTypes(networkname.AmoyChainName, borTypes)
	snapcfg.RegisterKnownTypes(networkname.BorMainnetChainName, borTypes)
}

var (
	BorEvents = snaptype.RegisterType(
		snaptype.Enums.BorEvents,
		snaptype.Versions{
			Current:      1, //2,
			MinSupported: 1,
		},
		snaptype.RangeExtractorFunc(
			func(ctx context.Context, blockFrom, blockTo uint64, _ snaptype.FirstKeyGetter, db kv.RoDB, chainConfig *chain.Config, collect func([]byte) error, workers int, lvl log.Lvl, logger log.Logger) (uint64, error) {
				logEvery := time.NewTicker(20 * time.Second)
				defer logEvery.Stop()

				from := hexutility.EncodeTs(blockFrom)
				var first bool = true
				var prevBlockNum uint64
				var startEventId uint64
				var lastEventId uint64
				if err := kv.BigChunks(db, kv.BorEventNums, from, func(tx kv.Tx, blockNumBytes, eventIdBytes []byte) (bool, error) {
					blockNum := binary.BigEndian.Uint64(blockNumBytes)
					if first {
						startEventId = binary.BigEndian.Uint64(eventIdBytes)
						first = false
						prevBlockNum = blockNum
					} else if blockNum != prevBlockNum {
						endEventId := binary.BigEndian.Uint64(eventIdBytes)
						blockHash, e := rawdb.ReadCanonicalHash(tx, prevBlockNum)
						if e != nil {
							return false, e
						}
						if e := extractEventRange(startEventId, endEventId, tx, prevBlockNum, blockHash, collect); e != nil {
							return false, e
						}
						startEventId = endEventId
						prevBlockNum = blockNum
					}
					if blockNum >= blockTo {
						return false, nil
					}
					lastEventId = binary.BigEndian.Uint64(eventIdBytes)
					select {
					case <-ctx.Done():
						return false, ctx.Err()
					case <-logEvery.C:
						var m runtime.MemStats
						if lvl >= log.LvlInfo {
							dbg.ReadMemStats(&m)
						}
						logger.Log(lvl, "[bor snapshots] Dumping bor events", "block num", blockNum,
							"alloc", common.ByteCount(m.Alloc), "sys", common.ByteCount(m.Sys),
						)
					default:
					}
					return true, nil
				}); err != nil {
					return 0, err
				}
				if lastEventId > startEventId {
					if err := db.View(ctx, func(tx kv.Tx) error {
						blockHash, e := rawdb.ReadCanonicalHash(tx, prevBlockNum)
						if e != nil {
							return e
						}
						return extractEventRange(startEventId, lastEventId+1, tx, prevBlockNum, blockHash, collect)
					}); err != nil {
						return 0, err
					}
				}

				return lastEventId, nil
			}),
		[]snaptype.Index{snaptype.Indexes.BorTxnHash},
		snaptype.IndexBuilderFunc(
			func(ctx context.Context, sn snaptype.FileInfo, chainConfig *chain.Config, tmpDir string, p *background.Progress, lvl log.Lvl, logger log.Logger) (err error) {
				defer func() {
					if rec := recover(); rec != nil {
						err = fmt.Errorf("BorEventsIdx: at=%d-%d, %v, %s", sn.From, sn.To, rec, dbg.Stack())
					}
				}()
				// Calculate how many records there will be in the index
				d, err := seg.NewDecompressor(sn.Path)
				if err != nil {
					return err
				}
				defer d.Close()
				g := d.MakeGetter()
				var blockNumBuf [length.BlockNum]byte
				var first bool = true
				word := make([]byte, 0, 4096)
				var blockCount int
				var baseEventId uint64
				for g.HasNext() {
					word, _ = g.Next(word[:0])
					if first || !bytes.Equal(blockNumBuf[:], word[length.Hash:length.Hash+length.BlockNum]) {
						blockCount++
						copy(blockNumBuf[:], word[length.Hash:length.Hash+length.BlockNum])
					}
					if first {
						baseEventId = binary.BigEndian.Uint64(word[length.Hash+length.BlockNum : length.Hash+length.BlockNum+8])
						first = false
					}
					select {
					case <-ctx.Done():
						return ctx.Err()
					default:
					}
				}

				rs, err := recsplit.NewRecSplit(recsplit.RecSplitArgs{
					KeyCount:   blockCount,
					Enums:      blockCount > 0,
					BucketSize: 2000,
					LeafSize:   8,
					TmpDir:     tmpDir,
					IndexFile:  filepath.Join(sn.Dir(), snaptype.IdxFileName(sn.Version, sn.From, sn.To, snaptype.Enums.BorEvents.String())),
					BaseDataID: baseEventId,
				}, logger)
				if err != nil {
					return err
				}
				rs.LogLvl(log.LvlDebug)

				defer d.EnableMadvNormal().DisableReadAhead()
			RETRY:
				g.Reset(0)
				first = true
				var i, offset, nextPos uint64
				for g.HasNext() {
					word, nextPos = g.Next(word[:0])
					i++
					if first || !bytes.Equal(blockNumBuf[:], word[length.Hash:length.Hash+length.BlockNum]) {
						if err = rs.AddKey(word[:length.Hash], offset); err != nil {
							return err
						}
						copy(blockNumBuf[:], word[length.Hash:length.Hash+length.BlockNum])
					}
					if first {
						first = false
					}
					select {
					case <-ctx.Done():
						return ctx.Err()
					default:
					}
					offset = nextPos
				}
				if err = rs.Build(ctx); err != nil {
					if errors.Is(err, recsplit.ErrCollision) {
						logger.Info("Building recsplit. Collision happened. It's ok. Restarting with another salt...", "err", err)
						rs.ResetNextSalt()
						goto RETRY
					}
					return err
				}

				return nil
			}))

	BorSpans = snaptype.RegisterType(
		snaptype.Enums.BorSpans,
		snaptype.Versions{
			Current:      1, //2,
			MinSupported: 1,
		},
		snaptype.RangeExtractorFunc(
			func(ctx context.Context, blockFrom, blockTo uint64, firstKeyGetter snaptype.FirstKeyGetter, db kv.RoDB, chainConfig *chain.Config, collect func([]byte) error, workers int, lvl log.Lvl, logger log.Logger) (uint64, error) {
				spanFrom := uint64(heimdall.SpanIdAt(blockFrom))
				spanTo := uint64(heimdall.SpanIdAt(blockTo))
				return extractValueRange(ctx, kv.BorSpans, spanFrom, spanTo, db, chainConfig, collect, workers, lvl, logger)
			}),
		[]snaptype.Index{snaptype.Indexes.BorSpanId},
		snaptype.IndexBuilderFunc(
			func(ctx context.Context, sn snaptype.FileInfo, chainConfig *chain.Config, tmpDir string, p *background.Progress, lvl log.Lvl, logger log.Logger) (err error) {
				d, err := seg.NewDecompressor(sn.Path)

				if err != nil {
					return err
				}
				defer d.Close()

				baseSpanId := uint64(heimdall.SpanIdAt(sn.From))

				return buildValueIndex(ctx, sn, d, baseSpanId, chainConfig, tmpDir, p, lvl, logger)
			}),
	)

	BorCheckpoints = snaptype.RegisterType(
		snaptype.Enums.BorCheckpoints,
		snaptype.Versions{
			Current:      1, //2,
			MinSupported: 1,
		},
		snaptype.RangeExtractorFunc(
			func(ctx context.Context, blockFrom, blockTo uint64, firstKeyGetter snaptype.FirstKeyGetter, db kv.RoDB, chainConfig *chain.Config, collect func([]byte) error, workers int, lvl log.Lvl, logger log.Logger) (uint64, error) {
				checkpointFrom, err := heimdall.CheckpointIdAt(ctx, db, blockFrom)

				if err != nil {
					return 0, err
				}

				if blockFrom > 0 {
					if prevTo, err := heimdall.CheckpointIdAt(ctx, db, blockFrom-1); err == nil && prevTo == checkpointFrom {
						checkpointFrom++
					}
				}

				checkpointTo, err := heimdall.CheckpointIdAt(ctx, db, blockTo)

				if err != nil {
					return 0, err
				}

				return extractValueRange(ctx, kv.BorCheckpoints, uint64(checkpointFrom), uint64(checkpointTo), db, chainConfig, collect, workers, lvl, logger)
			}),
		[]snaptype.Index{snaptype.Indexes.BorCheckpointId},
		snaptype.IndexBuilderFunc(
			func(ctx context.Context, sn snaptype.FileInfo, chainConfig *chain.Config, tmpDir string, p *background.Progress, lvl log.Lvl, logger log.Logger) (err error) {
				d, err := seg.NewDecompressor(sn.Path)

				if err != nil {
					return err
				}
				defer d.Close()

				buf, _ := d.MakeGetter().Next(nil)
				var firstCheckpoint heimdall.Checkpoint

				if err = json.Unmarshal(buf, &firstCheckpoint); err != nil {
					return err
				}

				return buildValueIndex(ctx, sn, d, uint64(firstCheckpoint.Id), chainConfig, tmpDir, p, lvl, logger)
			}),
	)

	BorMilestones = snaptype.RegisterType(
		snaptype.Enums.BorMilestones,
		snaptype.Versions{
			Current:      1, //2,
			MinSupported: 1,
		},
		snaptype.RangeExtractorFunc(
			func(ctx context.Context, blockFrom, blockTo uint64, firstKeyGetter snaptype.FirstKeyGetter, db kv.RoDB, chainConfig *chain.Config, collect func([]byte) error, workers int, lvl log.Lvl, logger log.Logger) (uint64, error) {
				milestoneFrom, err := heimdall.MilestoneIdAt(ctx, db, blockFrom)

				if err != nil && !errors.Is(err, heimdall.ErrMilestoneNotFound) {
					return 0, err
				}

				if milestoneFrom > 0 && blockFrom > 0 {
					if prevTo, err := heimdall.MilestoneIdAt(ctx, db, blockFrom-1); err == nil && prevTo == milestoneFrom {
						milestoneFrom++
					}
				}

				milestoneTo, err := heimdall.MilestoneIdAt(ctx, db, blockTo)

				if err != nil && !errors.Is(err, heimdall.ErrMilestoneNotFound) {
					return 0, err
				}

				if milestoneTo < milestoneFrom {
					return 0, fmt.Errorf("end milestone: %d before start milestone: %d", milestoneTo, milestoneFrom)
				}

				return extractValueRange(ctx, kv.BorMilestones, uint64(milestoneFrom), uint64(milestoneTo), db, chainConfig, collect, workers, lvl, logger)
			}),
		[]snaptype.Index{snaptype.Indexes.BorMilestoneId},
		snaptype.IndexBuilderFunc(
			func(ctx context.Context, sn snaptype.FileInfo, chainConfig *chain.Config, tmpDir string, p *background.Progress, lvl log.Lvl, logger log.Logger) (err error) {
				d, err := seg.NewDecompressor(sn.Path)

				if err != nil {
					return err
				}
				defer d.Close()

				gg := d.MakeGetter()

				var firstMilestoneId uint64

				if gg.HasNext() {
					buf, _ := gg.Next(nil)
					if len(buf) > 0 {
						var firstMilestone heimdall.Milestone
						if err = json.Unmarshal(buf, &firstMilestone); err != nil {
							return err
						}
						firstMilestoneId = uint64(firstMilestone.Id)
					}
				}

				return buildValueIndex(ctx, sn, d, firstMilestoneId, chainConfig, tmpDir, p, lvl, logger)
			}),
	)

	BorSnapshotTypes = []snaptype.Type{BorEvents, BorSpans, BorCheckpoints, BorMilestones}
)

func extractValueRange(ctx context.Context, table string, valueFrom, valueTo uint64, db kv.RoDB, chainConfig *chain.Config, collect func([]byte) error, workers int, lvl log.Lvl, logger log.Logger) (uint64, error) {
	logEvery := time.NewTicker(20 * time.Second)
	defer logEvery.Stop()

	if err := kv.BigChunks(db, table, hexutility.EncodeTs(valueFrom), func(tx kv.Tx, idBytes, valueBytes []byte) (bool, error) {
		id := binary.BigEndian.Uint64(idBytes)
		if id >= valueTo {
			return false, nil
		}
		if e := collect(valueBytes); e != nil {
			return false, e
		}
		select {
		case <-ctx.Done():
			return false, ctx.Err()
		case <-logEvery.C:
			var m runtime.MemStats
			if lvl >= log.LvlInfo {
				dbg.ReadMemStats(&m)
			}
			logger.Log(lvl, "[bor snapshots] Dumping bor values", "id", id,
				"alloc", common.ByteCount(m.Alloc), "sys", common.ByteCount(m.Sys),
			)
		default:
		}
		return true, nil
	}); err != nil {
		return valueTo, err
	}
	return valueTo, nil
}

func buildValueIndex(ctx context.Context, sn snaptype.FileInfo, d *seg.Decompressor, baseId uint64, chainConfig *chain.Config, tmpDir string, p *background.Progress, lvl log.Lvl, logger log.Logger) (err error) {
	defer func() {
		if rec := recover(); rec != nil {
			err = fmt.Errorf("BorSpansIdx: at=%d-%d, %v, %s", sn.From, sn.To, rec, dbg.Stack())
		}
	}()

	rs, err := recsplit.NewRecSplit(recsplit.RecSplitArgs{
		KeyCount:   d.Count(),
		Enums:      d.Count() > 0,
		BucketSize: 2000,
		LeafSize:   8,
		TmpDir:     tmpDir,
		IndexFile:  filepath.Join(sn.Dir(), sn.Type.IdxFileName(sn.Version, sn.From, sn.To)),
		BaseDataID: uint64(baseId),
	}, logger)
	if err != nil {
		return err
	}
	rs.LogLvl(log.LvlDebug)

	defer d.EnableMadvNormal().DisableReadAhead()
RETRY:

	g := d.MakeGetter()
	var i, offset, nextPos uint64
	var key [8]byte
	for g.HasNext() {
		nextPos, _ = g.Skip()
		binary.BigEndian.PutUint64(key[:], i)
		i++
		if err = rs.AddKey(key[:], offset); err != nil {
			return err
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
		offset = nextPos
	}
	if err = rs.Build(ctx); err != nil {
		if errors.Is(err, recsplit.ErrCollision) {
			logger.Info("Building recsplit. Collision happened. It's ok. Restarting with another salt...", "err", err)
			rs.ResetNextSalt()
			goto RETRY
		}
		return err
	}

	return nil
}

func extractEventRange(startEventId, endEventId uint64, tx kv.Tx, blockNum uint64, blockHash common.Hash, collect func([]byte) error) error {
	var blockNumBuf [8]byte
	var eventIdBuf [8]byte
	txnHash := bor_types.ComputeBorTxHash(blockNum, blockHash)
	binary.BigEndian.PutUint64(blockNumBuf[:], blockNum)
	for eventId := startEventId; eventId < endEventId; eventId++ {
		binary.BigEndian.PutUint64(eventIdBuf[:], eventId)
		event, err := tx.GetOne(kv.BorEvents, eventIdBuf[:])
		if err != nil {
			return err
		}
		snapshotRecord := make([]byte, len(event)+length.Hash+length.BlockNum+8)
		copy(snapshotRecord, txnHash[:])
		copy(snapshotRecord[length.Hash:], blockNumBuf[:])
		binary.BigEndian.PutUint64(snapshotRecord[length.Hash+length.BlockNum:], eventId)
		copy(snapshotRecord[length.Hash+length.BlockNum+8:], event)
		if err := collect(snapshotRecord); err != nil {
			return err
		}
	}
	return nil
}
