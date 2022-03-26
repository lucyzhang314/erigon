package stagedsync

import (
	"bytes"
	"context"
	"encoding/binary"
	"fmt"

	libcommon "github.com/ledgerwatch/erigon-lib/common"
	"github.com/ledgerwatch/erigon-lib/gointerfaces"
	"github.com/ledgerwatch/erigon-lib/gointerfaces/remote"
	types2 "github.com/ledgerwatch/erigon-lib/gointerfaces/types"
	"github.com/ledgerwatch/erigon-lib/kv"
	common2 "github.com/ledgerwatch/erigon/common"
	"github.com/ledgerwatch/erigon/common/dbutils"
	"github.com/ledgerwatch/erigon/core/rawdb"
	"github.com/ledgerwatch/erigon/core/types"
	"github.com/ledgerwatch/erigon/ethdb/cbor"
	"github.com/ledgerwatch/erigon/params"
	"github.com/ledgerwatch/log/v3"
)

type FinishCfg struct {
	db     kv.RwDB
	tmpDir string
	log    log.Logger
}

func StageFinishCfg(db kv.RwDB, tmpDir string, logger log.Logger) FinishCfg {
	return FinishCfg{
		db:     db,
		log:    logger,
		tmpDir: tmpDir,
	}
}

func FinishForward(s *StageState, tx kv.RwTx, cfg FinishCfg, initialCycle bool) error {
	useExternalTx := tx != nil
	if !useExternalTx {
		var err error
		tx, err = cfg.db.BeginRw(context.Background())
		if err != nil {
			return err
		}
		defer tx.Rollback()
	}

	var executionAt uint64
	var err error
	if executionAt, err = s.ExecutionAt(tx); err != nil {
		return err
	}
	if executionAt <= s.BlockNumber {
		return nil
	}

	rawdb.WriteHeadBlockHash(tx, rawdb.ReadHeadHeaderHash(tx))
	err = s.Update(tx, executionAt)
	if err != nil {
		return err
	}

	if initialCycle {
		if err := params.SetErigonVersion(tx, params.VersionKeyFinished); err != nil {
			return err
		}
	}

	if !useExternalTx {
		if err := tx.Commit(); err != nil {
			return err
		}
	}

	return nil
}

func UnwindFinish(u *UnwindState, tx kv.RwTx, cfg FinishCfg, ctx context.Context) (err error) {
	useExternalTx := tx != nil
	if !useExternalTx {
		tx, err = cfg.db.BeginRw(ctx)
		if err != nil {
			return err
		}
		defer tx.Rollback()
	}

	if err = u.Done(tx); err != nil {
		return err
	}
	if !useExternalTx {
		if err = tx.Commit(); err != nil {
			return err
		}
	}
	return nil
}

func PruneFinish(u *PruneState, tx kv.RwTx, cfg FinishCfg, ctx context.Context) (err error) {
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

func NotifyNewHeaders(ctx context.Context, finishStageBeforeSync uint64, finishStageAfterSync uint64, unwindTo *uint64, notifier ChainEventNotifier, tx kv.Tx) error {
	if notifier == nil {
		log.Trace("RPC Daemon notification channel not set. No headers notifications will be sent")
		return nil
	}
	// Notify all headers we have (either canonical or not) in a maximum range span of 1024
	var notifyFrom uint64
	var isUnwind bool
	if unwindTo != nil && *unwindTo != 0 && (*unwindTo) < finishStageBeforeSync {
		notifyFrom = *unwindTo
		isUnwind = true
	} else {
		heightSpan := finishStageAfterSync - finishStageBeforeSync
		if heightSpan > 1024 {
			heightSpan = 1024
		}
		notifyFrom = finishStageAfterSync - heightSpan
	}
	notifyFrom++

	var notifyTo uint64
	var headersRlp [][]byte
	if err := tx.ForEach(kv.Headers, dbutils.EncodeBlockNumber(notifyFrom), func(k, headerRLP []byte) error {
		if len(headerRLP) == 0 {
			return nil
		}
		notifyTo = binary.BigEndian.Uint64(k)
		headersRlp = append(headersRlp, common2.CopyBytes(headerRLP))
		return libcommon.Stopped(ctx.Done())
	}); err != nil {
		log.Error("RPC Daemon notification failed", "err", err)
		return err
	}
	notifier.OnNewHeader(headersRlp)
	log.Info("RPC Daemon notified of new headers", "from", notifyFrom-1, "to", notifyTo)
	logs, err := ReadLogs(tx, notifyFrom, isUnwind)
	if err != nil {
		return err
	}
	notifier.OnLogs(logs)

	return nil
}

func ReadLogs(tx kv.Tx, from uint64, isUnwind bool) ([]*remote.SubscribeLogsReply, error) {
	logs, err := tx.Cursor(kv.Log)
	if err != nil {
		return nil, err
	}
	defer logs.Close()
	reply := make([]*remote.SubscribeLogsReply, 0)
	reader := bytes.NewReader(nil)

	for k, v, err := logs.Seek(dbutils.LogKey(from, 0)); k != nil; k, v, err = logs.Next() {
		if err != nil {
			return nil, err
		}
		blockNum := binary.BigEndian.Uint64(k[:8])
		var ll types.Logs
		reader.Reset(v)
		if err := cbor.Unmarshal(&ll, reader); err != nil {
			return nil, fmt.Errorf("receipt unmarshal failed: %w, blocl=%d", err, blockNum)
		}

		for _, l := range ll {
			r := &remote.SubscribeLogsReply{
				Address:          gointerfaces.ConvertAddressToH160(l.Address),
				BlockHash:        gointerfaces.ConvertHashToH256(l.BlockHash),
				BlockNumber:      l.BlockNumber,
				Data:             l.Data,
				LogIndex:         uint64(l.Index),
				Topics:           make([]*types2.H256, 0, len(l.Topics)),
				TransactionHash:  gointerfaces.ConvertHashToH256(l.TxHash),
				TransactionIndex: uint64(l.TxIndex),
				Removed:          isUnwind,
			}
			for _, topic := range l.Topics {
				r.Topics = append(r.Topics, gointerfaces.ConvertHashToH256(topic))
			}
			reply = append(reply, r)
		}
	}

	return reply, nil
}
