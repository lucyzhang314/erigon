package stagedsync

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"math"
	"sync"
	"time"

	"github.com/holiman/uint256"
	libcommon "github.com/ledgerwatch/erigon-lib/common"
	"github.com/ledgerwatch/erigon-lib/common/length"
	"github.com/ledgerwatch/erigon-lib/etl"
	"github.com/ledgerwatch/erigon-lib/kv"
	"github.com/ledgerwatch/erigon/common"
	"github.com/ledgerwatch/erigon/common/dbutils"
	"github.com/ledgerwatch/erigon/common/debug"
	"github.com/ledgerwatch/erigon/core/rawdb"
	"github.com/ledgerwatch/erigon/core/types"
	"github.com/ledgerwatch/erigon/eth/stagedsync/stages"
	"github.com/ledgerwatch/erigon/ethdb/prune"
	"github.com/ledgerwatch/erigon/params"
	"github.com/ledgerwatch/erigon/turbo/snapshotsync"
	"github.com/ledgerwatch/erigon/turbo/snapshotsync/snapshothashes"
	"github.com/ledgerwatch/log/v3"
	"github.com/ledgerwatch/secp256k1"
)

type SendersCfg struct {
	db                kv.RwDB
	batchSize         int
	blockSize         int
	bufferSize        int
	numOfGoroutines   int
	readChLen         int
	tmpdir            string
	prune             prune.Mode
	chainConfig       *params.ChainConfig
	blockRetire       *snapshotsync.BlockRetire
	snapshotHashesCfg *snapshothashes.Config
}

func StageSendersCfg(db kv.RwDB, chainCfg *params.ChainConfig, tmpdir string, prune prune.Mode, br *snapshotsync.BlockRetire) SendersCfg {
	const sendersBatchSize = 10000
	const sendersBlockSize = 4096

	return SendersCfg{
		db:                db,
		batchSize:         sendersBatchSize,
		blockSize:         sendersBlockSize,
		bufferSize:        (sendersBlockSize * 10 / 20) * 10000, // 20*4096
		numOfGoroutines:   secp256k1.NumOfContexts(),            // we can only be as parallels as our crypto library supports,
		readChLen:         4,
		tmpdir:            tmpdir,
		chainConfig:       chainCfg,
		prune:             prune,
		blockRetire:       br,
		snapshotHashesCfg: snapshothashes.KnownConfig(chainCfg.ChainName),
	}
}

func SpawnRecoverSendersStage(cfg SendersCfg, s *StageState, u Unwinder, tx kv.RwTx, toBlock uint64, ctx context.Context) error {
	quitCh := ctx.Done()
	useExternalTx := tx != nil
	if !useExternalTx {
		var err error
		tx, err = cfg.db.BeginRw(context.Background())
		if err != nil {
			return err
		}
		defer tx.Rollback()
	}

	prevStageProgress, errStart := stages.GetStageProgress(tx, stages.Bodies)
	if errStart != nil {
		return errStart
	}

	var to = prevStageProgress
	if toBlock > 0 {
		to = min(prevStageProgress, toBlock)
	}
	if to <= s.BlockNumber {
		return nil
	}
	logPrefix := s.LogPrefix()
	if to > s.BlockNumber+16 {
		log.Info(fmt.Sprintf("[%s] Started", logPrefix), "from", s.BlockNumber, "to", to)
	}

	logEvery := time.NewTicker(30 * time.Second)
	defer logEvery.Stop()

	canonical := make([]common.Hash, to-s.BlockNumber)
	currentHeaderIdx := uint64(0)

	canonicalC, err := tx.Cursor(kv.HeaderCanonical)
	if err != nil {
		return err
	}
	defer canonicalC.Close()

	startFrom := s.BlockNumber + 1
	if cfg.blockRetire.Snapshots() != nil && startFrom < cfg.blockRetire.Snapshots().BlocksAvailable() {
		startFrom = cfg.blockRetire.Snapshots().BlocksAvailable()
	}

	for k, v, err := canonicalC.Seek(dbutils.EncodeBlockNumber(startFrom)); k != nil; k, v, err = canonicalC.Next() {
		if err != nil {
			return err
		}
		if err := libcommon.Stopped(quitCh); err != nil {
			return err
		}

		if currentHeaderIdx >= to-s.BlockNumber { // if header stage is ehead of body stage
			break
		}

		copy(canonical[currentHeaderIdx][:], v)
		currentHeaderIdx++

		select {
		default:
		case <-logEvery.C:
			log.Info(fmt.Sprintf("[%s] Preload headers", logPrefix), "block_number", binary.BigEndian.Uint64(k))
		}
	}
	log.Trace(fmt.Sprintf("[%s] Read canonical hashes", logPrefix), "amount", len(canonical))

	jobs := make(chan *senderRecoveryJob, cfg.batchSize)
	out := make(chan *senderRecoveryJob, cfg.batchSize)
	wg := new(sync.WaitGroup)
	wg.Add(cfg.numOfGoroutines)
	ctx, cancelWorkers := context.WithCancel(context.Background())
	defer cancelWorkers()
	for i := 0; i < cfg.numOfGoroutines; i++ {
		go func(threadNo int) {
			defer debug.LogPanic()
			defer wg.Done()
			// each goroutine gets it's own crypto context to make sure they are really parallel
			recoverSenders(ctx, logPrefix, secp256k1.ContextForThread(threadNo), cfg.chainConfig, jobs, out, quitCh)
		}(i)
	}

	collectorSenders := etl.NewCollector(logPrefix, cfg.tmpdir, etl.NewSortableBuffer(etl.BufferOptimalSize))
	defer collectorSenders.Close()

	errCh := make(chan senderRecoveryError)
	go func() {
		defer debug.LogPanic()
		defer close(errCh)
		defer cancelWorkers()
		var ok bool
		var j *senderRecoveryJob
		for {
			select {
			case <-quitCh:
				return
			case <-logEvery.C:
				n := s.BlockNumber
				if j != nil {
					n += uint64(j.index)
				}
				log.Info(fmt.Sprintf("[%s] Recovery", logPrefix), "block_number", n)
			case j, ok = <-out:
				if !ok {
					return
				}
				if j.err != nil {
					errCh <- senderRecoveryError{err: j.err, blockNumber: j.blockNumber, blockHash: j.blockHash}
					return
				}

				k := make([]byte, 4)
				binary.BigEndian.PutUint32(k, uint32(j.index))
				index := int(binary.BigEndian.Uint32(k))
				if err := collectorSenders.Collect(dbutils.BlockBodyKey(s.BlockNumber+uint64(index)+1, canonical[index]), j.senders); err != nil {
					errCh <- senderRecoveryError{err: j.err}
					return
				}
			}
		}
	}()

	var minBlockNum uint64 = math.MaxUint64
	var minBlockHash common.Hash
	var minBlockErr error
	handleRecoverErr := func(recErr senderRecoveryError) error {
		if recErr.blockHash == (common.Hash{}) {
			return recErr.err
		}

		if recErr.blockNumber < minBlockNum {
			minBlockNum = recErr.blockNumber
			minBlockHash = recErr.blockHash
			minBlockErr = recErr.err
		}
		return nil
	}

	bodiesC, err := tx.Cursor(kv.BlockBody)
	if err != nil {
		return err
	}
	defer bodiesC.Close()

Loop:
	for k, _, err := bodiesC.Seek(dbutils.EncodeBlockNumber(s.BlockNumber + 1)); k != nil; k, _, err = bodiesC.Next() {
		if err != nil {
			return err
		}
		if err := libcommon.Stopped(quitCh); err != nil {
			return err
		}

		blockNumber := binary.BigEndian.Uint64(k[:8])
		blockHash := common.BytesToHash(k[8:])
		if blockNumber > to {
			break
		}

		if canonical[blockNumber-s.BlockNumber-1] != blockHash {
			// non-canonical case
			continue
		}
		body := rawdb.ReadCanonicalBodyWithTransactions(tx, blockHash, blockNumber)

		select {
		case recoveryErr := <-errCh:
			if recoveryErr.err != nil {
				cancelWorkers()
				if err := handleRecoverErr(recoveryErr); err != nil {
					return err
				}
				break Loop
			}
		case jobs <- &senderRecoveryJob{body: body, key: k, blockNumber: blockNumber, blockHash: blockHash, index: int(blockNumber - s.BlockNumber - 1)}:
		}
	}

	close(jobs)
	wg.Wait()
	close(out)
	for recoveryErr := range errCh {
		if recoveryErr.err != nil {
			cancelWorkers()
			if err := handleRecoverErr(recoveryErr); err != nil {
				return err
			}
		}
	}
	if minBlockErr != nil {
		log.Error(fmt.Sprintf("[%s] Error recovering senders for block %d %x): %v", logPrefix, minBlockNum, minBlockHash, minBlockErr))
		if to > s.BlockNumber {
			u.UnwindTo(minBlockNum-1, minBlockHash)
		}
	} else {
		if err := collectorSenders.Load(tx, kv.Senders, etl.IdentityLoadFunc, etl.TransformArgs{
			Quit: quitCh,
			LogDetailsLoad: func(k, v []byte) (additionalLogArguments []interface{}) {
				return []interface{}{"block", binary.BigEndian.Uint64(k)}
			},
		}); err != nil {
			return err
		}
		if err = s.Update(tx, to); err != nil {
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

type senderRecoveryError struct {
	err         error
	blockNumber uint64
	blockHash   common.Hash
}

type senderRecoveryJob struct {
	body        *types.Body
	key         []byte
	senders     []byte
	blockHash   common.Hash
	blockNumber uint64
	index       int
	err         error
}

func recoverSenders(ctx context.Context, logPrefix string, cryptoContext *secp256k1.Context, config *params.ChainConfig, in, out chan *senderRecoveryJob, quit <-chan struct{}) {
	var job *senderRecoveryJob
	var ok bool
	for {
		select {
		case job, ok = <-in:
			if !ok {
				return
			}
			if job == nil {
				return
			}
		case <-ctx.Done():
			return
		case <-quit:
			return
		}

		body := job.body
		signer := types.MakeSigner(config, job.blockNumber)
		job.senders = make([]byte, len(body.Transactions)*length.Addr)
		for i, tx := range body.Transactions {
			from, err := signer.SenderWithContext(cryptoContext, tx)
			if err != nil {
				job.err = fmt.Errorf("%s: error recovering sender for tx=%x, %w", logPrefix, tx.Hash(), err)
				break
			}
			copy(job.senders[i*length.Addr:], from[:])
		}

		// prevent sending to close channel
		if err := libcommon.Stopped(quit); err != nil {
			job.err = err
		} else if err = libcommon.Stopped(ctx.Done()); err != nil {
			job.err = err
		}
		out <- job

		if errors.Is(job.err, libcommon.ErrStopped) {
			return
		}
	}
}

func UnwindSendersStage(s *UnwindState, tx kv.RwTx, cfg SendersCfg, ctx context.Context) (err error) {
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

func PruneSendersStage(s *PruneState, tx kv.RwTx, cfg SendersCfg, ctx context.Context) (err error) {
	logEvery := time.NewTicker(logInterval)
	defer logEvery.Stop()
	to := cfg.prune.TxIndex.PruneTo(s.ForwardProgress)
	useExternalTx := tx != nil
	if !useExternalTx {
		tx, err = cfg.db.BeginRw(ctx)
		if err != nil {
			return err
		}
		defer tx.Rollback()
	}

	if cfg.blockRetire.Snapshots() != nil && cfg.blockRetire.Snapshots().Cfg().RetireEnabled {
		if err := retireBlocks(s, tx, cfg, ctx); err != nil {
			return fmt.Errorf("retireBlocks: %w", err)
		}
	} else {
		if err = PruneTable(tx, kv.Senders, s.LogPrefix(), to, logEvery, ctx); err != nil {
			return err
		}
	}

	if !useExternalTx {
		if err = tx.Commit(); err != nil {
			return err
		}
	}
	return nil
}

func retireBlocks(s *PruneState, tx kv.RwTx, cfg SendersCfg, ctx context.Context) (err error) {
	if cfg.blockRetire.Working() {
		return nil
	}
	if res := cfg.blockRetire.Result(); res != nil {
		if res.Err != nil {
			return fmt.Errorf("[%s] retire blocks last error: %w", s.LogPrefix(), res.Err)
		}
	}

	canDeleteTo := cfg.blockRetire.CanDeleteTo(s.ForwardProgress)
	if err := rawdb.DeleteAncientBlocks(tx, canDeleteTo, 1_000); err != nil {
		return nil
	}

	// TODO: remove this check for the release
	if err := cfg.blockRetire.Snapshots().EnsureExpectedBlocksAreAvailable(cfg.snapshotHashesCfg); err != nil {
		return err
	}
	blockFrom, blockTo, ok := cfg.blockRetire.CanRetire(s.ForwardProgress)
	if !ok {
		return nil
	}

	chainID, _ := uint256.FromBig(cfg.chainConfig.ChainID)
	cfg.blockRetire.RetireBlocksInBackground(ctx, blockFrom, blockTo, *chainID, log.LvlDebug)

	return nil
}
