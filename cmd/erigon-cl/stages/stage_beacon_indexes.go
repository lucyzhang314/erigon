package stages

import (
	"context"
	"fmt"
	"time"

	"github.com/ledgerwatch/erigon-lib/etl"
	"github.com/ledgerwatch/erigon-lib/kv"
	"github.com/ledgerwatch/erigon/cl/utils"
	"github.com/ledgerwatch/erigon/cmd/erigon-cl/core/rawdb"
	"github.com/ledgerwatch/erigon/eth/stagedsync"
	"github.com/ledgerwatch/erigon/eth/stagedsync/stages"
	"github.com/ledgerwatch/log/v3"
)

type StageBeaconIndexesCfg struct {
	db     kv.RwDB
	tmpdir string
}

func StageBeaconIndexes(db kv.RwDB, tmpdir string) StageBeaconIndexesCfg {
	return StageBeaconIndexesCfg{
		db:     db,
		tmpdir: tmpdir,
	}
}

// SpawnStageBeaconsForward spawn the beacon forward stage
func SpawnStageBeaconIndexes(cfg StageBeaconIndexesCfg, s *stagedsync.StageState, tx kv.RwTx, ctx context.Context) error {
	useExternalTx := tx != nil
	var err error
	if !useExternalTx {
		tx, err = cfg.db.BeginRw(ctx)
		if err != nil {
			return err
		}
		defer tx.Rollback()
	}
	progress := s.BlockNumber
	if progress != 0 {
		progress++
	}
	slotToRootCollector := etl.NewCollector(s.LogPrefix(), cfg.tmpdir, etl.NewSortableBuffer(etl.BufferOptimalSize))
	rootToSlotCollector := etl.NewCollector(s.LogPrefix(), cfg.tmpdir, etl.NewSortableBuffer(etl.BufferOptimalSize))

	endSlot, err := stages.GetStageProgress(tx, stages.BeaconBlocks)
	if err != nil {
		return err
	}
	logInterval := time.NewTicker(logIntervalTime)
	defer logInterval.Stop()

	for slot := progress; slot <= endSlot; slot++ {
		block, _, eth1Hash, eth2Hash, err := rawdb.ReadBeaconBlockForStorage(tx, slot)
		if err != nil {
			return err
		}
		// Missed proposal are absent slot
		if block == nil {
			continue
		}
		slotBytes := utils.Uint32ToBytes4(uint32(slot))
		// Collect slot => root indexes
		if err := slotToRootCollector.Collect(slotBytes[:], eth1Hash[:]); err != nil {
			return err
		}
		if err := slotToRootCollector.Collect(slotBytes[:], eth2Hash[:]); err != nil {
			return err
		}
		if err := slotToRootCollector.Collect(slotBytes[:], block.Block.StateRoot[:]); err != nil {
			return err
		}
		// Collect root indexes => slot
		if err := rootToSlotCollector.Collect(eth1Hash[:], slotBytes[:]); err != nil {
			return err
		}
		if err := rootToSlotCollector.Collect(eth2Hash[:], slotBytes[:]); err != nil {
			return err
		}
		if err := rootToSlotCollector.Collect(block.Block.StateRoot[:], slotBytes[:]); err != nil {
			return err
		}
		select {
		case <-logInterval.C:
			log.Info(fmt.Sprintf("[%s] Progress", s.LogPrefix()), "slot", slot, "remaining", endSlot-slot)
		default:
		}
	}
	if !useExternalTx {
		if err = tx.Commit(); err != nil {
			return err
		}
	}
	return nil
}
