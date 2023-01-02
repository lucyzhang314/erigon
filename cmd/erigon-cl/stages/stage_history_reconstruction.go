package stages

import (
	"context"
	"fmt"
	"time"

	"github.com/ledgerwatch/erigon-lib/etl"
	"github.com/ledgerwatch/erigon-lib/kv"
	"github.com/ledgerwatch/erigon/cl/clparams"
	"github.com/ledgerwatch/erigon/cl/cltypes"
	"github.com/ledgerwatch/erigon/cl/utils"
	"github.com/ledgerwatch/erigon/cmd/erigon-cl/core/rawdb"
	"github.com/ledgerwatch/erigon/cmd/erigon-cl/core/state"
	"github.com/ledgerwatch/erigon/cmd/erigon-cl/execution_client"
	"github.com/ledgerwatch/erigon/cmd/erigon-cl/network"
	"github.com/ledgerwatch/erigon/eth/stagedsync"
	"github.com/ledgerwatch/log/v3"
)

type StageHistoryReconstructionCfg struct {
	db              kv.RwDB
	genesisCfg      *clparams.GenesisConfig
	beaconCfg       *clparams.BeaconChainConfig
	downloader      *network.BackwardBeaconDownloader
	state           *state.BeaconState
	executionClient *execution_client.ExecutionClient
	tmpdir          string
}

const RecEnabled = true

var DestinationSlot = 0

const logIntervalTime = 30 * time.Second

func StageHistoryReconstruction(db kv.RwDB, downloader *network.BackwardBeaconDownloader, genesisCfg *clparams.GenesisConfig, beaconCfg *clparams.BeaconChainConfig, state *state.BeaconState, tmpdir string, executionClient *execution_client.ExecutionClient) StageHistoryReconstructionCfg {
	return StageHistoryReconstructionCfg{
		db:              db,
		genesisCfg:      genesisCfg,
		beaconCfg:       beaconCfg,
		downloader:      downloader,
		state:           state,
		tmpdir:          tmpdir,
		executionClient: executionClient,
	}
}

// SpawnStageBeaconsForward spawn the beacon forward stage
func SpawnStageHistoryReconstruction(cfg StageHistoryReconstructionCfg, s *stagedsync.StageState, tx kv.RwTx, ctx context.Context) error {
	// This stage must be done only once.
	progress := s.BlockNumber
	if progress != 0 || !RecEnabled {
		return nil
	}

	useExternalTx := tx != nil
	var err error
	if !useExternalTx {
		tx, err = cfg.db.BeginRw(ctx)
		if err != nil {
			return err
		}
		defer tx.Rollback()
	}
	blockRoot, err := cfg.state.BlockRoot()
	if err != nil {
		return err
	}
	DestinationSlot = int(utils.GetCurrentSlot(cfg.genesisCfg.GenesisTime, cfg.beaconCfg.SecondsPerSlot)) - 1_000_000
	// ETL collectors for attestations + beacon blocks
	beaconBlocksCollector := etl.NewCollector(s.LogPrefix(), cfg.tmpdir, etl.NewSortableBuffer(etl.BufferOptimalSize))
	defer beaconBlocksCollector.Close()
	attestationsCollector := etl.NewCollector(s.LogPrefix(), cfg.tmpdir, etl.NewSortableBuffer(etl.BufferOptimalSize))
	defer attestationsCollector.Close()
	executionPayloadsCollector := etl.NewCollector(s.LogPrefix(), cfg.tmpdir, etl.NewSortableBuffer(etl.BufferOptimalSize))
	defer executionPayloadsCollector.Close()

	log.Info(fmt.Sprintf("[%s] Reconstructing", s.LogPrefix()), "from", cfg.state.LatestBlockHeader().Slot, "to", DestinationSlot)
	// Setup slot and block root
	cfg.downloader.SetSlotToDownload(cfg.state.LatestBlockHeader().Slot)
	cfg.downloader.SetExpectedRoot(blockRoot)
	foundLatestEth1ValidHash := false
	// Set up onNewBlock callback
	cfg.downloader.SetOnNewBlock(func(blk *cltypes.SignedBeaconBlock) (finished bool, err error) {
		slot := blk.Block.Slot
		// Collect attestations
		encodedAttestations := cltypes.EncodeAttestationsForStorage(blk.Block.Body.Attestations)
		if err := attestationsCollector.Collect(rawdb.EncodeNumber(slot), encodedAttestations); err != nil {
			return false, err
		}
		// Collect beacon blocks
		encodedBeaconBlock, err := blk.EncodeForStorage()
		if err != nil {
			return false, err
		}
		if err := beaconBlocksCollector.Collect(rawdb.EncodeNumber(slot), encodedBeaconBlock); err != nil {
			return false, err
		}
		// Collect Execution Payloads
		if cfg.executionClient != nil && blk.Version() >= clparams.BellatrixVersion && !foundLatestEth1ValidHash {
			payload := blk.Block.Body.ExecutionPayload
			if foundLatestEth1ValidHash, err = cfg.executionClient.IsCanonical(payload.BlockHash); err != nil {
				return false, err
			}
			if foundLatestEth1ValidHash {
				return slot == uint64(DestinationSlot), nil
			}
			encodedPayload, err := payload.MarshalSSZ()
			if err != nil {
				return false, err
			}
			if err := executionPayloadsCollector.Collect(rawdb.EncodeNumber(slot), encodedPayload); err != nil {
				return false, err
			}
		}
		// will arbitratly stop at slot 5.1M for testing reasons
		return /*slot == uint64(DestinationSlot), nil*/ foundLatestEth1ValidHash, nil
	})
	prevProgress := cfg.downloader.Progress()

	logInterval := time.NewTicker(30 * time.Second)
	finishCh := make(chan struct{})
	// Start logging thread
	go func() {
		for {
			select {
			case <-logInterval.C:
				currProgress := cfg.downloader.Progress()
				speed := (float64(prevProgress) - float64(currProgress)) / (float64(logIntervalTime) / float64(time.Second))
				prevProgress = currProgress
				peerCount, err := cfg.downloader.Peers()
				if err != nil {
					return
				}
				log.Info(fmt.Sprintf("[%s] Backwards downloading phase", s.LogPrefix()),
					"progress", currProgress,
					"remaining", currProgress-uint64(DestinationSlot),
					"blk/sec", fmt.Sprintf("%.1f", speed),
					"peers", peerCount,
				)
			case <-finishCh:
				return
			case <-ctx.Done():

			}
		}
	}()
	for !cfg.downloader.Finished() {
		cfg.downloader.RequestMore()
	}
	close(finishCh)
	if err := attestationsCollector.Load(tx, kv.Attestetations, etl.IdentityLoadFunc, etl.TransformArgs{Quit: context.Background().Done()}); err != nil {
		return err
	}
	if err := beaconBlocksCollector.Load(tx, kv.BeaconBlocks, etl.IdentityLoadFunc, etl.TransformArgs{Quit: context.Background().Done()}); err != nil {
		return err
	}
	// Send in ordered manner EL blocks to Execution Layer
	if err := executionPayloadsCollector.Load(tx, kv.BeaconBlocks, func(k, v []byte, table etl.CurrentTableReader, next etl.LoadNextFunc) error {
		payload := &cltypes.ExecutionPayload{}
		if err := payload.UnmarshalSSZ(v); err != nil {
			return err
		}
		if err := cfg.executionClient.InsertExecutionPayload(payload); err != nil {
			return err
		}
		return next(k, nil, nil)
	}, etl.TransformArgs{Quit: context.Background().Done()}); err != nil {
		return err
	}
	if err := s.Update(tx, 1); err != nil {
		return err
	}

	if !useExternalTx {
		if err = tx.Commit(); err != nil {
			return err
		}
	}
	return nil
}
