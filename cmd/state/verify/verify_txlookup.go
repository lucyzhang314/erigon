package verify

import (
	"bytes"
	"context"
	"fmt"
	"math/big"
	"os"
	"os/signal"
	"time"

	"github.com/ledgerwatch/erigon/common"
	"github.com/ledgerwatch/erigon/core/rawdb"
	"github.com/ledgerwatch/erigon/ethdb/kv"
	"github.com/ledgerwatch/erigon/ethdb/olddb"
	"github.com/ledgerwatch/erigon/log"
)

func ValidateTxLookups(chaindata string) error {
	db := olddb.MustOpen(chaindata)
	tx, err := db.BeginRo(context.Background())
	if err != nil {
		return err
	}
	defer tx.Rollback()

	ch := make(chan os.Signal, 1)
	quitCh := make(chan struct{})
	signal.Notify(ch, os.Interrupt)
	go func() {
		<-ch
		close(quitCh)
	}()
	t := time.Now()
	defer func() {
		log.Info("Validation ended", "it took", time.Since(t))
	}()
	var blockNum uint64
	iterations := 0
	var interrupt bool
	// Validation Process
	blockBytes := big.NewInt(0)
	for !interrupt {
		if err := common.Stopped(quitCh); err != nil {
			return err
		}
		blockHash, err := rawdb.ReadCanonicalHash(tx, blockNum)
		if err != nil {
			return err
		}
		body := rawdb.ReadBody(tx, blockHash, blockNum)

		if body == nil {
			log.Error("Empty body", "blocknum", blockNum)
			break
		}
		blockBytes.SetUint64(blockNum)
		bn := blockBytes.Bytes()

		for _, txn := range body.Transactions {
			val, err := tx.GetOne(kv.TxLookup, txn.Hash().Bytes())
			iterations++
			if iterations%100000 == 0 {
				log.Info("Validated", "entries", iterations, "number", blockNum)

			}
			if !bytes.Equal(val, bn) {
				if err != nil {
					panic(err)
				}
				panic(fmt.Sprintf("Validation process failed(%d). Expected %b, got %b", iterations, bn, val))
			}
		}
		blockNum++
	}
	return nil
}
