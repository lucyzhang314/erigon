package commands

import (
	"context"
	"fmt"
	"math/big"
	"time"

	"github.com/holiman/uint256"
	jsoniter "github.com/json-iterator/go"
	"github.com/ledgerwatch/erigon/common"
	"github.com/ledgerwatch/erigon/common/hexutil"
	"github.com/ledgerwatch/erigon/common/math"
	"github.com/ledgerwatch/erigon/consensus/ethash"
	"github.com/ledgerwatch/erigon/core"
	"github.com/ledgerwatch/erigon/core/rawdb"
	"github.com/ledgerwatch/erigon/core/state"
	"github.com/ledgerwatch/erigon/core/types"
	"github.com/ledgerwatch/erigon/core/vm"
	"github.com/ledgerwatch/erigon/eth/tracers"
	"github.com/ledgerwatch/erigon/ethdb"
	"github.com/ledgerwatch/erigon/internal/ethapi"
	"github.com/ledgerwatch/erigon/rpc"
	"github.com/ledgerwatch/erigon/turbo/rpchelper"
	"github.com/ledgerwatch/erigon/turbo/transactions"
	"github.com/ledgerwatch/log/v3"
)

// TraceBlockByNumber implements debug_traceBlockByNumber. Returns Geth style block traces.
func (api *PrivateDebugAPIImpl) TraceBlockByNumber(ctx context.Context, blockNum rpc.BlockNumber, config *tracers.TraceConfig, stream *jsoniter.Stream) error {
	return api.traceBlock(ctx, rpc.BlockNumberOrHashWithNumber(blockNum), config, stream)
}

// TraceBlockByHash implements debug_traceBlockByHash. Returns Geth style block traces.
func (api *PrivateDebugAPIImpl) TraceBlockByHash(ctx context.Context, hash common.Hash, config *tracers.TraceConfig, stream *jsoniter.Stream) error {
	return api.traceBlock(ctx, rpc.BlockNumberOrHashWithHash(hash, true), config, stream)
}

func (api *PrivateDebugAPIImpl) traceBlock(ctx context.Context, blockNrOrHash rpc.BlockNumberOrHash, config *tracers.TraceConfig, stream *jsoniter.Stream) error {
	tx, err := api.db.BeginRo(ctx)
	if err != nil {
		stream.WriteNil()
		return err
	}
	defer tx.Rollback()
	var (
		block    *types.Block
		number   rpc.BlockNumber
		numberOk bool
		hash     common.Hash
		hashOk   bool
	)
	if number, numberOk = blockNrOrHash.Number(); numberOk {
		block, err = api.blockByRPCNumber(number, tx)
	} else if hash, hashOk = blockNrOrHash.Hash(); hashOk {
		block, err = api.blockByHashWithSenders(tx, hash)
	} else {
		return fmt.Errorf("invalid arguments; neither block nor hash specified")
	}

	if err != nil {
		stream.WriteNil()
		return err
	}

	if block == nil {
		if numberOk {
			return fmt.Errorf("invalid arguments; block with number %d not found", number)
		}
		return fmt.Errorf("invalid arguments; block with hash %v not found", hash)
	}

	chainConfig, err := api.chainConfig(tx)
	if err != nil {
		stream.WriteNil()
		return err
	}

	contractHasTEVM := func(contractHash common.Hash) (bool, error) { return false, nil }
	if api.TevmEnabled {
		contractHasTEVM = ethdb.GetHasTEVM(tx)
	}

	getHeader := func(hash common.Hash, number uint64) *types.Header {
		h, e := api._blockReader.Header(ctx, tx, hash, number)
		if e != nil {
			log.Error("getHeader error", "number", number, "hash", hash, "err", e)
		}
		return h
	}

	_, blockCtx, _, ibs, reader, err := transactions.ComputeTxEnv(ctx, block, chainConfig, getHeader, contractHasTEVM, ethash.NewFaker(), tx, block.Hash(), 0)
	if err != nil {
		stream.WriteNil()
		return err
	}

	signer := types.MakeSigner(chainConfig, block.NumberU64())
	rules := chainConfig.Rules(block.NumberU64())
	stream.WriteArrayStart()
	for idx, tx := range block.Transactions() {
		select {
		default:
		case <-ctx.Done():
			stream.WriteNil()
			return ctx.Err()
		}
		ibs.Prepare(tx.Hash(), block.Hash(), idx)
		msg, _ := tx.AsMessage(*signer, block.BaseFee(), rules)
		txCtx := vm.TxContext{
			TxHash:   tx.Hash(),
			Origin:   msg.From(),
			GasPrice: msg.GasPrice().ToBig(),
		}

		transactions.TraceTx(ctx, msg, blockCtx, txCtx, ibs, config, chainConfig, stream)
		_ = ibs.FinalizeTx(rules, reader)
		if idx != len(block.Transactions())-1 {
			stream.WriteMore()
		}
		stream.Flush()
	}
	stream.WriteArrayEnd()
	stream.Flush()
	return nil
}

// TraceTransaction implements debug_traceTransaction. Returns Geth style transaction traces.
func (api *PrivateDebugAPIImpl) TraceTransaction(ctx context.Context, hash common.Hash, config *tracers.TraceConfig, stream *jsoniter.Stream) error {
	tx, err := api.db.BeginRo(ctx)
	if err != nil {
		stream.WriteNil()
		return err
	}
	defer tx.Rollback()
	// Retrieve the transaction and assemble its EVM context
	blockNum, ok, err := api.txnLookup(ctx, tx, hash)
	if err != nil {
		stream.WriteNil()
		return err
	}
	if !ok {
		stream.WriteNil()
		return nil
	}
	block, err := api.blockByNumberWithSenders(tx, blockNum)
	if err != nil {
		stream.WriteNil()
		return err
	}
	if block == nil {
		stream.WriteNil()
		return nil
	}
	blockHash := block.Hash()
	var txnIndex uint64
	var txn types.Transaction
	for i, transaction := range block.Transactions() {
		if transaction.Hash() == hash {
			txnIndex = uint64(i)
			txn = transaction
			break
		}
	}
	if txn == nil {
		var borTx types.Transaction
		borTx, _, _, _, err = rawdb.ReadBorTransaction(tx, hash)
		if err != nil {
			stream.WriteNil()
			return err
		}

		if borTx != nil {
			stream.WriteNil()
			return nil
		}
		stream.WriteNil()
		return fmt.Errorf("transaction %#x not found", hash)
	}
	chainConfig, err := api.chainConfig(tx)
	if err != nil {
		stream.WriteNil()
		return err
	}

	getHeader := func(hash common.Hash, number uint64) *types.Header {
		return rawdb.ReadHeader(tx, hash, number)
	}
	contractHasTEVM := func(contractHash common.Hash) (bool, error) { return false, nil }
	if api.TevmEnabled {
		contractHasTEVM = ethdb.GetHasTEVM(tx)
	}
	msg, blockCtx, txCtx, ibs, _, err := transactions.ComputeTxEnv(ctx, block, chainConfig, getHeader, contractHasTEVM, ethash.NewFaker(), tx, blockHash, txnIndex)
	if err != nil {
		stream.WriteNil()
		return err
	}
	// Trace the transaction and return
	return transactions.TraceTx(ctx, msg, blockCtx, txCtx, ibs, config, chainConfig, stream)
}

func (api *PrivateDebugAPIImpl) TraceCall(ctx context.Context, args ethapi.CallArgs, blockNrOrHash rpc.BlockNumberOrHash, config *tracers.TraceConfig, stream *jsoniter.Stream) error {
	dbtx, err := api.db.BeginRo(ctx)
	if err != nil {
		stream.WriteNil()
		return err
	}
	defer dbtx.Rollback()

	chainConfig, err := api.chainConfig(dbtx)
	if err != nil {
		stream.WriteNil()
		return err
	}

	blockNumber, hash, latest, err := rpchelper.GetBlockNumber(blockNrOrHash, dbtx, api.filters)
	if err != nil {
		stream.WriteNil()
		return err
	}
	var stateReader state.StateReader
	if latest {
		cacheView, err := api.stateCache.View(ctx, dbtx)
		if err != nil {
			return err
		}
		stateReader = state.NewCachedReader2(cacheView, dbtx)
	} else {
		stateReader = state.NewPlainState(dbtx, blockNumber)
	}
	header := rawdb.ReadHeader(dbtx, hash, blockNumber)
	if header == nil {
		stream.WriteNil()
		return fmt.Errorf("block %d(%x) not found", blockNumber, hash)
	}
	ibs := state.New(stateReader)

	if config != nil && config.StateOverrides != nil {
		if err := config.StateOverrides.Override(ibs); err != nil {
			return err
		}
	}

	var baseFee *uint256.Int
	if header != nil && header.BaseFee != nil {
		var overflow bool
		baseFee, overflow = uint256.FromBig(header.BaseFee)
		if overflow {
			return fmt.Errorf("header.BaseFee uint256 overflow")
		}
	}
	msg, err := args.ToMessage(api.GasCap, baseFee)
	if err != nil {
		return err
	}

	contractHasTEVM := func(contractHash common.Hash) (bool, error) { return false, nil }
	if api.TevmEnabled {
		contractHasTEVM = ethdb.GetHasTEVM(dbtx)
	}
	blockCtx, txCtx := transactions.GetEvmContext(msg, header, blockNrOrHash.RequireCanonical, dbtx, contractHasTEVM, api._blockReader)
	// Trace the transaction and return
	return transactions.TraceTx(ctx, msg, blockCtx, txCtx, ibs, config, chainConfig, stream)
}

func (api *PrivateDebugAPIImpl) TraceCallMany(ctx context.Context, bundles []Bundle, simulateContext StateContext, config *tracers.TraceConfig, stream *jsoniter.Stream) error {
	var (
		hash               common.Hash
		replayTransactions types.Transactions
		evm                *vm.EVM
		blockCtx           vm.BlockContext
		txCtx              vm.TxContext
		overrideBlockHash  map[uint64]common.Hash
		baseFee            uint256.Int
	)

	overrideBlockHash = make(map[uint64]common.Hash)
	tx, err := api.db.BeginRo(ctx)
	if err != nil {
		stream.WriteNil()
		return err
	}
	defer tx.Rollback()
	chainConfig, err := api.chainConfig(tx)
	if err != nil {
		stream.WriteNil()
		return err
	}
	if len(bundles) == 0 {
		stream.WriteNil()
		return fmt.Errorf("empty bundles")
	}
	empty := true
	for _, bundle := range bundles {
		if len(bundle.Transactions) != 0 {
			empty = false
		}
	}

	if empty {
		stream.WriteNil()
		return fmt.Errorf("empty bundles")
	}

	defer func(start time.Time) { log.Trace("Tracing CallMany finished", "runtime", time.Since(start)) }(time.Now())

	blockNum, hash, _, err := rpchelper.GetBlockNumber(simulateContext.BlockNumber, tx, api.filters)
	if err != nil {
		stream.WriteNil()
		return err
	}

	block, err := api.blockByNumberWithSenders(tx, blockNum)
	if err != nil {
		stream.WriteNil()
		return err
	}

	// -1 is a default value for transaction index.
	// If it's -1, we will try to replay every single transaction in that block
	transactionIndex := -1

	if simulateContext.TransactionIndex != nil {
		transactionIndex = *simulateContext.TransactionIndex
	}

	if transactionIndex == -1 {
		transactionIndex = len(block.Transactions())
	}

	replayTransactions = block.Transactions()[:transactionIndex]

	stateReader, err := rpchelper.CreateStateReader(ctx, tx, rpc.BlockNumberOrHashWithNumber(rpc.BlockNumber(blockNum-1)), api.filters, api.stateCache)

	if err != nil {
		stream.WriteNil()
		return err
	}

	st := state.New(stateReader)

	parent := block.Header()

	if parent == nil {
		stream.WriteNil()
		return fmt.Errorf("block %d(%x) not found", blockNum, hash)
	}

	// Get a new instance of the EVM
	signer := types.MakeSigner(chainConfig, blockNum)
	rules := chainConfig.Rules(blockNum)

	contractHasTEVM := func(contractHash common.Hash) (bool, error) { return false, nil }

	if api.TevmEnabled {
		contractHasTEVM = ethdb.GetHasTEVM(tx)
	}

	getHash := func(i uint64) common.Hash {
		if hash, ok := overrideBlockHash[i]; ok {
			return hash
		}
		hash, err := rawdb.ReadCanonicalHash(tx, i)
		if err != nil {
			log.Debug("Can't get block hash by number", "number", i, "only-canonical", true)
		}
		return hash
	}

	if parent.BaseFee != nil {
		baseFee.SetFromBig(parent.BaseFee)
	}

	blockCtx = vm.BlockContext{
		CanTransfer:     core.CanTransfer,
		Transfer:        core.Transfer,
		GetHash:         getHash,
		ContractHasTEVM: contractHasTEVM,
		Coinbase:        parent.Coinbase,
		BlockNumber:     parent.Number.Uint64(),
		Time:            parent.Time,
		Difficulty:      new(big.Int).Set(parent.Difficulty),
		GasLimit:        parent.GasLimit,
		BaseFee:         &baseFee,
	}

	evm = vm.NewEVM(blockCtx, txCtx, st, chainConfig, vm.Config{Debug: false})

	// Setup the gas pool (also for unmetered requests)
	// and apply the message.
	gp := new(core.GasPool).AddGas(math.MaxUint64)
	for _, txn := range replayTransactions {
		msg, err := txn.AsMessage(*signer, nil, rules)
		if err != nil {
			stream.WriteNil()
			return err
		}
		txCtx = core.NewEVMTxContext(msg)
		evm = vm.NewEVM(blockCtx, txCtx, evm.IntraBlockState(), chainConfig, vm.Config{Debug: false})
		// Execute the transaction message
		_, err = core.ApplyMessage(evm, msg, gp, true /* refunds */, false /* gasBailout */)
		if err != nil {
			stream.WriteNil()
			return err
		}

	}

	// after replaying the txns, we want to overload the state
	if config.StateOverrides != nil {
		err = config.StateOverrides.Override(evm.IntraBlockState().(*state.IntraBlockState))
		if err != nil {
			stream.WriteNil()
			return err
		}
	}

	stream.WriteArrayStart()
	for bundle_index, bundle := range bundles {
		stream.WriteArrayStart()
		// first change blockContext
		blockHeaderOverride(&blockCtx, bundle.BlockOverride, overrideBlockHash)
		for txn_index, txn := range bundle.Transactions {
			if txn.Gas == nil || *(txn.Gas) == 0 {
				txn.Gas = (*hexutil.Uint64)(&api.GasCap)
			}
			msg, err := txn.ToMessage(api.GasCap, blockCtx.BaseFee)
			if err != nil {
				stream.WriteNil()
				return err
			}
			txCtx = core.NewEVMTxContext(msg)
			ibs := evm.IntraBlockState().(*state.IntraBlockState)
			ibs.Prepare(common.Hash{}, parent.Hash(), txn_index)
			err = transactions.TraceTx(ctx, msg, blockCtx, txCtx, evm.IntraBlockState(), config, chainConfig, stream)

			if err != nil {
				stream.WriteNil()
				return err
			}

			if txn_index < len(bundle.Transactions)-1 {
				stream.WriteMore()
			}
		}
		stream.WriteArrayEnd()

		if bundle_index < len(bundles)-1 {
			stream.WriteMore()
		}
		blockCtx.BlockNumber++
		blockCtx.Time++
	}
	stream.WriteArrayEnd()
	return nil
}
