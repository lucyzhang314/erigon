package stagedsync

import (
	"context"
	"fmt"
	"github.com/ledgerwatch/erigon/common/dbutils"
	"github.com/ledgerwatch/erigon/common/etl"
	"github.com/ledgerwatch/erigon/eth/ethconfig"
	"github.com/ledgerwatch/erigon/eth/stagedsync/stages"
	"github.com/ledgerwatch/erigon/ethdb"
	"github.com/ledgerwatch/erigon/ethdb/kv"
	"github.com/ledgerwatch/erigon/log"
	"github.com/ledgerwatch/erigon/turbo/snapshotsync"
)

type SnapshotStateCfg struct {
	enabled          bool
	db               ethdb.RwKV
	snapshotDir      string
	tmpDir           string
	client           *snapshotsync.Client
	snapshotMigrator *snapshotsync.SnapshotMigrator
}

func StageSnapshotStateCfg(db ethdb.RwKV, snapshot ethconfig.Snapshot, tmpDir string, client *snapshotsync.Client, snapshotMigrator *snapshotsync.SnapshotMigrator) SnapshotStateCfg {
	return SnapshotStateCfg{
		enabled:          snapshot.Enabled && snapshot.Mode.State,
		db:               db,
		snapshotDir:      snapshot.Dir,
		client:           client,
		snapshotMigrator: snapshotMigrator,
		tmpDir:           tmpDir,
	}
}

func SpawnStateSnapshotGenerationStage(s *StageState, tx ethdb.RwTx, cfg SnapshotStateCfg, ctx context.Context, initialSync bool, epochSize uint64) (err error) {
	if !initialSync {
		return nil
	}
	roTX, err := cfg.db.BeginRo(ctx)
	if err != nil {
		return err
	}
	defer roTX.Rollback()

	to, err := stages.GetStageProgress(roTX, stages.Execution)
	if err != nil {
		return fmt.Errorf("%w", err)
	}

	//it's too early for snapshot
	if to < epochSize {
		return nil
	}
	currentSnapshotBlock, err := stages.GetStageProgress(roTX, stages.CreateStateSnapshot)
	if err != nil {
		return fmt.Errorf("%w", err)
	}
	snapshotBlock := snapshotsync.CalculateEpoch(to, epochSize)
	if snapshotBlock <= currentSnapshotBlock {
		return nil
	}
	roTX.Rollback()

	//prelimary checks finished. we can start migration.
	tmpDB := cfg.db.(*kv.SnapshotKV).TempDB()
	if tmpDB != nil {
		log.Error("Empty tmp db")
		defer func() {
			//recover tmp db in case of error
			if err != nil {
				cfg.db.(*kv.SnapshotKV).SetTempDB(tmpDB, snapshotsync.StateSnapshotBuckets)
			}
		}()
	}

	//get rid of block after epoch block
	cfg.db.(*kv.SnapshotKV).SetTempDB(nil, nil)
	mainDBTX, err := cfg.db.BeginRw(ctx)
	if err != nil {
		return err
	}
	defer mainDBTX.Rollback()

	//collect whole state snapshot
	plainStateCollector := etl.NewCollector(cfg.tmpDir, etl.NewSortableBuffer(etl.BufferOptimalSize))
	codeCollector := etl.NewCollector(cfg.tmpDir, etl.NewSortableBuffer(etl.BufferOptimalSize))
	contractCodeBucketCollector := etl.NewCollector(cfg.tmpDir, etl.NewSortableBuffer(etl.BufferOptimalSize))
	err = mainDBTX.ForEach(dbutils.PlainStateBucket, []byte{}, func(k, v []byte) error {
		return plainStateCollector.Collect(k, v)
	})
	if err != nil {
		return err
	}
	err = mainDBTX.ForEach(dbutils.CodeBucket, []byte{}, func(k, v []byte) error {
		return codeCollector.Collect(k, v)
	})
	if err != nil {
		return err
	}
	err = mainDBTX.ForEach(dbutils.PlainContractCodeBucket, []byte{}, func(k, v []byte) error {
		return contractCodeBucketCollector.Collect(k, v)
	})
	if err != nil {
		return err
	}

	snapshotPath := snapshotsync.SnapshotName(cfg.tmpDir, "state", snapshotBlock)
	//todo change tmp dir to snapshots folder
	snKV, err := snapshotsync.CreateStateSnapshot(ctx, snapshotPath)
	if err != nil {
		return err
	}

	snRwTX, err := snKV.BeginRw(context.Background())
	if err != nil {
		return err
	}

	err = plainStateCollector.Load("plain state", snRwTX, dbutils.PlainStateBucket, etl.IdentityLoadFunc, etl.TransformArgs{
		Quit: ctx.Done(),
	})
	if err != nil {
		return err
	}

	err = codeCollector.Load("codes", snRwTX, dbutils.CodeBucket, etl.IdentityLoadFunc, etl.TransformArgs{
		Quit: ctx.Done(),
	})
	if err != nil {
		return err
	}

	err = contractCodeBucketCollector.Load("code hashes", snRwTX, dbutils.PlainContractCodeBucket, etl.IdentityLoadFunc, etl.TransformArgs{
		Quit: ctx.Done(),
	})
	if err != nil {
		return err
	}

	err = snRwTX.Commit()
	if err != nil {
		return err
	}
	snKV.Close()
	//snapshot creation finished

	err = mainDBTX.ClearBucket(dbutils.PlainStateBucket)
	if err != nil {
		return err
	}
	err = mainDBTX.ClearBucket(dbutils.CodeBucket)
	if err != nil {
		return err
	}
	err = mainDBTX.ClearBucket(dbutils.PlainContractCodeBucket)
	if err != nil {
		return err
	}

	if tmpDB != nil {
		tmpDBRoTX, err := tmpDB.BeginRo(context.Background())
		if err != nil {
			return err
		}
		defer tmpDBRoTX.Rollback()
		migrateBucket := func(from ethdb.Tx, to ethdb.RwTx, bucket string) error {
			return from.ForEach(bucket, []byte{}, func(k, v []byte) error {
				return to.Put(bucket, k, v)
			})
		}

		err = migrateBucket(tmpDBRoTX, mainDBTX, dbutils.PlainStateBucket)
		if err != nil {
			return err
		}

		err = migrateBucket(tmpDBRoTX, mainDBTX, dbutils.PlainContractCodeBucket)
		if err != nil {
			return err
		}

		err = migrateBucket(tmpDBRoTX, mainDBTX, dbutils.ContractCodeBucket)
		if err != nil {
			return err
		}
	}

	if err = s.Update(mainDBTX, snapshotBlock); err != nil {
		return err
	}
	stateSnapshot, err := snapshotsync.OpenStateSnapshot(snapshotPath)
	if err != nil {
		return err
	}
	err = mainDBTX.Commit()
	if err != nil {
		return err
	}

	cfg.db.(*kv.SnapshotKV).UpdateSnapshots("state", stateSnapshot, make(chan struct{}))

	if tmpDB != nil {
		go func() {
			tmpDB.Close()
			//todo remove tmp db

		}()
	}
	return nil
}

func UnwindStateSnapshotGenerationStage(s *UnwindState, tx ethdb.RwTx, cfg SnapshotStateCfg, ctx context.Context) (err error) {
	useExternalTx := tx != nil
	if !useExternalTx {
		tx, err = cfg.db.BeginRw(ctx)
		if err != nil {
			return err
		}
		defer tx.Rollback()
	}

	if err = s.Done(tx); err != nil {
		return err
	}
	if !useExternalTx {
		if err = tx.Commit(); err != nil {
			return err
		}
	}
	return nil
}

func PruneStateSnapshotGenerationStage(s *PruneState, tx ethdb.RwTx, cfg SnapshotStateCfg, ctx context.Context) (err error) {
	useExternalTx := tx != nil
	if !useExternalTx {
		tx, err = cfg.db.BeginRw(ctx)
		if err != nil {
			return err
		}
		defer tx.Rollback()
	}

	if !useExternalTx {
		if err = tx.Commit(); err != nil {
			return err
		}
	}
	return nil
}
