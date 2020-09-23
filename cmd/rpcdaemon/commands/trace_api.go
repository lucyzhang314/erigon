package commands

import (
	"bytes"
	"context"
	"fmt"
	"sort"

	"github.com/ledgerwatch/turbo-geth/cmd/rpcdaemon/cli"
	"github.com/ledgerwatch/turbo-geth/common"
	"github.com/ledgerwatch/turbo-geth/common/hexutil"
	"github.com/ledgerwatch/turbo-geth/core/rawdb"
	"github.com/ledgerwatch/turbo-geth/core/types"
	"github.com/ledgerwatch/turbo-geth/eth"
	"github.com/ledgerwatch/turbo-geth/ethdb"
	"github.com/ledgerwatch/turbo-geth/rpc"
	"github.com/ledgerwatch/turbo-geth/turbo/adapter"
	"github.com/ledgerwatch/turbo-geth/turbo/transactions"
)

// TraceAPI RPC interface into tracing API
type TraceAPI interface {
	// Ad-hoc
	// Call(ctx context.Context, req TraceFilterRequest) ([]interface{}, error)
	// CallMany(ctx context.Context, req TraceFilterRequest) ([]interface{}, error)
	// RawTransaction(ctx context.Context, req TraceFilterRequest) ([]interface{}, error)
	// ReplayBlockTransactions(ctx context.Context, req TraceFilterRequest) ([]interface{}, error)
	// ReplayTransaction(ctx context.Context, req TraceFilterRequest) ([]interface{}, error)

	// Filtering
	Block(ctx context.Context, blockNr rpc.BlockNumber) ([]interface{}, error)
	Filter(ctx context.Context, req TraceFilterRequest) ([]interface{}, error)
	Get(ctx context.Context, txHash common.Hash, txIndicies []hexutil.Uint64) (interface{}, error)
	Transaction(ctx context.Context, txHash common.Hash) ([]interface{}, error)

	// Custom (turbo geth exclusive)
	BlockReward(ctx context.Context, blockNr rpc.BlockNumber) (Issuance, error)
	UncleReward(ctx context.Context, blockNr rpc.BlockNumber) (Issuance, error)
	Issuance(ctx context.Context, blockNr rpc.BlockNumber) (Issuance, error)
}

// TraceAPIImpl is implementation of the TraceAPI interface based on remote Db access
type TraceAPIImpl struct {
	db        ethdb.KV
	dbReader  ethdb.Getter
	maxTraces uint64
	traceType string
}

// NewTraceAPI returns NewTraceAPI instance
func NewTraceAPI(db ethdb.KV, dbReader ethdb.Getter, cfg *cli.Flags) *TraceAPIImpl {
	return &TraceAPIImpl{
		db:        db,
		dbReader:  dbReader,
		maxTraces: cfg.MaxTraces,
		traceType: cfg.TraceType,
	}
}

// Filter Implements trace_filter
func (api *TraceAPIImpl) Filter(ctx context.Context, req TraceFilterRequest) ([]interface{}, error) {
	var filteredTransactionsHash []common.Hash
	var maxTracesCount uint64
	var offset uint64
	var skipped uint64

	sort.Slice(req.FromAddress, func(i int, j int) bool {
		return bytes.Compare(req.FromAddress[i].Bytes(), req.FromAddress[j].Bytes()) == -1
	})

	sort.Slice(req.ToAddress, func(i int, j int) bool {
		return bytes.Compare(req.ToAddress[i].Bytes(), req.ToAddress[j].Bytes()) == -1
	})

	var fromBlock uint64
	var toBlock uint64
	if req.FromBlock == nil {
		fromBlock = 0
	} else {
		fromBlock = uint64(*req.FromBlock)
	}

	if req.ToBlock == nil {
		headNumber := rawdb.ReadHeaderNumber(api.dbReader, rawdb.ReadHeadHeaderHash(api.dbReader))
		toBlock = *headNumber
	} else {
		toBlock = uint64(*req.ToBlock)
	}

	if fromBlock > toBlock {
		return nil, fmt.Errorf("invalid parameters: toBlock must be greater than fromBlock")
	}

	if req.Count == nil {
		maxTracesCount = api.maxTraces
	} else {
		maxTracesCount = *req.Count
	}

	if req.After == nil {
		offset = 0
	} else {
		offset = *req.After
	}

	if err := api.db.View(ctx, func(tx ethdb.Tx) error {
		if req.FromAddress != nil || req.ToAddress != nil { // use address history index to retrieve matching transactions
			var historyFilter []*common.Address
			isFromAddress := req.FromAddress != nil
			if isFromAddress {
				historyFilter = req.FromAddress
			} else {
				historyFilter = req.ToAddress
			}

			for _, addr := range historyFilter {

				addrBytes := addr.Bytes()
				blockNumbers, err := retrieveHistory(tx, addr, fromBlock, toBlock)
				if err != nil {
					return err
				}

				for _, num := range blockNumbers {

					block := rawdb.ReadBlockByNumber(api.dbReader, num)
					senders := rawdb.ReadSenders(api.dbReader, block.Hash(), num)
					txs := block.Transactions()
					for i, tx := range txs {
						if uint64(len(filteredTransactionsHash)) == maxTracesCount {
							if uint64(len(filteredTransactionsHash)) == api.maxTraces {
								return fmt.Errorf("too many traces found")
							}
							return nil
						}

						var to *common.Address
						if tx.To() == nil {
							to = &common.Address{}
						} else {
							to = tx.To()
						}

						if isFromAddress {
							if !isAddressInFilter(to, req.ToAddress) {
								continue
							}
							if bytes.Equal(senders[i].Bytes(), addrBytes) {
								filteredTransactionsHash = append(filteredTransactionsHash, tx.Hash())
							}
						} else if bytes.Equal(to.Bytes(), addrBytes) {
							if skipped < offset {
								skipped++
								continue
							}
							filteredTransactionsHash = append(filteredTransactionsHash, tx.Hash())
						}
					}
				}
			}
		} else if req.FromBlock != nil || req.ToBlock != nil { // iterate over blocks

			for blockNum := fromBlock; blockNum < toBlock+1; blockNum++ {
				block := rawdb.ReadBlockByNumber(api.dbReader, blockNum)
				blockTransactions := block.Transactions()
				for _, tx := range blockTransactions {
					if uint64(len(filteredTransactionsHash)) == maxTracesCount {
						if uint64(len(filteredTransactionsHash)) == api.maxTraces {
							return fmt.Errorf("too many traces found")
						}
						return nil
					}
					if skipped < offset {
						skipped++
						continue
					}
					filteredTransactionsHash = append(filteredTransactionsHash, tx.Hash())
				}
			}
		} else {
			return fmt.Errorf("invalid parameters")
		}
		return nil
	}); err != nil {
		return nil, err
	}
	getter := adapter.NewBlockGetter(api.dbReader)
	chainContext := adapter.NewChainContext(api.dbReader)
	genesisHash := rawdb.ReadBlockByNumber(api.dbReader, 0).Hash()
	chainConfig := rawdb.ReadChainConfig(api.dbReader, genesisHash)
	traceType := "callTracer"
	traces := []interface{}{}
	for _, txHash := range filteredTransactionsHash {
		_, blockHash, _, txIndex := rawdb.ReadTransaction(api.dbReader, txHash)
		msg, vmctx, ibs, _, err := transactions.ComputeTxEnv(ctx, getter, chainConfig, chainContext, api.db, blockHash, txIndex)
		if err != nil {
			return nil, err
		}
		trace, err := transactions.TraceTx(ctx, msg, vmctx, ibs, &eth.TraceConfig{Tracer: &traceType})
		if err != nil {
			return nil, err
		}
		traces = append(traces, trace)
	}
	return traces, nil
}

func (api *TraceAPIImpl) getBlockByRPCNumber(blockNr rpc.BlockNumber) (*types.Block, error) {
	blockNum, err := getBlockNumber(blockNr, api.dbReader)
	if err != nil {
		return nil, err
	}
	return rawdb.ReadBlockByNumber(api.dbReader, blockNum), nil
}
