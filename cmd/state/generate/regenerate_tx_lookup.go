package generate

import (
	"context"
	"os"
	"os/signal"
	"time"

	"github.com/ledgerwatch/turbo-geth/common/dbutils"
	"github.com/ledgerwatch/turbo-geth/eth/stagedsync"
	"github.com/ledgerwatch/turbo-geth/eth/stagedsync/stages"
	"github.com/ledgerwatch/turbo-geth/ethdb"
	"github.com/ledgerwatch/turbo-geth/log"
)

func RegenerateTxLookup(chaindata string) error {
	db := ethdb.MustOpen(chaindata)
	defer db.Close()
	if err := db.ClearBuckets(dbutils.TxLookupPrefix); err != nil {
		return err
	}
	tx, err := db.Begin(context.Background(), ethdb.RW)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	startTime := time.Now()
	ch := make(chan os.Signal, 1)
	quitCh := make(chan struct{})
	signal.Notify(ch, os.Interrupt)
	go func() {
		<-ch
		close(quitCh)
	}()

	lastExecutedBlock, err := stages.GetStageProgress(tx, stages.Execution)
	if err != nil {
		//There could be headers without block in the end
		log.Error("Cant get last executed block", "err", err)
	}
	log.Info("TxLookup generation started", "start time", startTime)
	err = stagedsync.TxLookupTransform("txlookup", tx.(ethdb.HasTx).Tx().(ethdb.RwTx), dbutils.EncodeBlockNumber(0), dbutils.EncodeBlockNumber(lastExecutedBlock+1), quitCh, stagedsync.StageTxLookupCfg(os.TempDir()))
	if err != nil {
		return err
	}
	log.Info("TxLookup index is successfully regenerated", "it took", time.Since(startTime))
	return nil
}
