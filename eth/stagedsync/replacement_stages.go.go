package stagedsync

import (
	"context"

	"github.com/ledgerwatch/turbo-geth/eth/stagedsync/stages"
	"github.com/ledgerwatch/turbo-geth/ethdb"
)

func ReplacementStages(ctx context.Context,
	sm ethdb.StorageMode,
	headers HeadersCfg,
	bodies BodiesCfg,
	senders SendersCfg,
	exec ExecuteBlockCfg,
	hashState HashStateCfg,
	trieCfg TrieCfg,
	history HistoryCfg,
	logIndex LogIndexCfg,
	callTraces CallTracesCfg,
	txLookup TxLookupCfg,
	txPool TxPoolCfg,
) StageBuilders {
	return []StageBuilder{
		{
			ID: stages.Headers,
			Build: func(world StageParameters) *Stage {
				return &Stage{
					ID:          stages.Headers,
					Description: "Download headers",
					ExecFunc: func(s *StageState, u Unwinder) error {
						var tx ethdb.RwTx
						if hasTx, ok := world.TX.(ethdb.HasTx); ok {
							tx = hasTx.Tx().(ethdb.RwTx)
						}
						return HeadersForward(s, u, ctx, tx, headers, world.InitialCycle)
					},
					UnwindFunc: func(u *UnwindState, s *StageState) error {
						var tx ethdb.RwTx
						if hasTx, ok := world.TX.(ethdb.HasTx); ok {
							tx = hasTx.Tx().(ethdb.RwTx)
						}
						return HeadersUnwind(u, s, tx, headers)
					},
				}
			},
		},
		{
			ID: stages.BlockHashes,
			Build: func(world StageParameters) *Stage {
				return &Stage{
					ID:          stages.BlockHashes,
					Description: "Write block hashes",
					ExecFunc: func(s *StageState, u Unwinder) error {
						return SpawnBlockHashStage(s, world.TX, world.TmpDir, ctx.Done())
					},
					UnwindFunc: func(u *UnwindState, s *StageState) error {
						return u.Done(world.TX)
					},
				}
			},
		},
		{
			ID: stages.Bodies,
			Build: func(world StageParameters) *Stage {
				return &Stage{
					ID:          stages.Bodies,
					Description: "Download block bodies",
					ExecFunc: func(s *StageState, u Unwinder) error {
						var tx ethdb.RwTx
						if hasTx, ok := world.TX.(ethdb.HasTx); ok {
							tx = hasTx.Tx().(ethdb.RwTx)
						}
						return BodiesForward(s, ctx, tx, bodies)
					},
					UnwindFunc: func(u *UnwindState, s *StageState) error {
						return u.Done(world.TX)
					},
				}
			},
		},
		{
			ID: stages.Senders,
			Build: func(world StageParameters) *Stage {
				return &Stage{
					ID:          stages.Senders,
					Description: "Recover senders from tx signatures",
					ExecFunc: func(s *StageState, u Unwinder) error {
						var tx ethdb.RwTx
						if hasTx, ok := world.TX.(ethdb.HasTx); ok {
							tx = hasTx.Tx().(ethdb.RwTx)
						}
						return SpawnRecoverSendersStage(senders, s, tx, 0, world.TmpDir, ctx.Done())
					},
					UnwindFunc: func(u *UnwindState, s *StageState) error {
						var tx ethdb.RwTx
						if hasTx, ok := world.TX.(ethdb.HasTx); ok {
							tx = hasTx.Tx().(ethdb.RwTx)
						}
						return UnwindSendersStage(u, s, tx, senders)
					},
				}
			},
		},
		{
			ID: stages.Execution,
			Build: func(world StageParameters) *Stage {
				return &Stage{
					ID:          stages.Execution,
					Description: "Execute blocks w/o hash checks",
					ExecFunc: func(s *StageState, u Unwinder) error {
						var tx ethdb.RwTx
						if hasTx, ok := world.TX.(ethdb.HasTx); ok {
							tx = hasTx.Tx().(ethdb.RwTx)
						}
						return SpawnExecuteBlocksStage(s, tx, 0, ctx.Done(), exec)
					},
					UnwindFunc: func(u *UnwindState, s *StageState) error {
						var tx ethdb.RwTx
						if hasTx, ok := world.TX.(ethdb.HasTx); ok {
							tx = hasTx.Tx().(ethdb.RwTx)
						}
						return UnwindExecutionStage(u, s, tx, ctx.Done(), exec)
					},
				}
			},
		},
		{
			ID: stages.HashState,
			Build: func(world StageParameters) *Stage {
				return &Stage{
					ID:          stages.HashState,
					Description: "Hash the key in the state",
					ExecFunc: func(s *StageState, u Unwinder) error {
						var tx ethdb.RwTx
						if hasTx, ok := world.TX.(ethdb.HasTx); ok {
							tx = hasTx.Tx().(ethdb.RwTx)
						}
						return SpawnHashStateStage(s, tx, hashState, ctx.Done())
					},
					UnwindFunc: func(u *UnwindState, s *StageState) error {
						var tx ethdb.RwTx
						if hasTx, ok := world.TX.(ethdb.HasTx); ok {
							tx = hasTx.Tx().(ethdb.RwTx)
						}
						return UnwindHashStateStage(u, s, tx, hashState, ctx.Done())
					},
				}
			},
		},
		{
			ID: stages.IntermediateHashes,
			Build: func(world StageParameters) *Stage {
				return &Stage{
					ID:          stages.IntermediateHashes,
					Description: "Generate intermediate hashes and computing state root",
					ExecFunc: func(s *StageState, u Unwinder) error {
						var tx ethdb.RwTx
						if hasTx, ok := world.TX.(ethdb.HasTx); ok {
							tx = hasTx.Tx().(ethdb.RwTx)
						}
						_, err := SpawnIntermediateHashesStage(s, u, tx, trieCfg, ctx.Done())
						return err
					},
					UnwindFunc: func(u *UnwindState, s *StageState) error {
						var tx ethdb.RwTx
						if hasTx, ok := world.TX.(ethdb.HasTx); ok {
							tx = hasTx.Tx().(ethdb.RwTx)
						}
						return UnwindIntermediateHashesStage(u, s, tx, trieCfg, ctx.Done())
					},
				}
			},
		},
		{
			ID: stages.AccountHistoryIndex,
			Build: func(world StageParameters) *Stage {
				return &Stage{
					ID:                  stages.AccountHistoryIndex,
					Description:         "Generate account history index",
					Disabled:            !sm.History,
					DisabledDescription: "Enable by adding `h` to --storage-mode",
					ExecFunc: func(s *StageState, u Unwinder) error {
						var tx ethdb.RwTx
						if hasTx, ok := world.TX.(ethdb.HasTx); ok {
							tx = hasTx.Tx().(ethdb.RwTx)
						}
						return SpawnAccountHistoryIndex(s, tx, history, ctx.Done())
					},
					UnwindFunc: func(u *UnwindState, s *StageState) error {
						var tx ethdb.RwTx
						if hasTx, ok := world.TX.(ethdb.HasTx); ok {
							tx = hasTx.Tx().(ethdb.RwTx)
						}
						return UnwindAccountHistoryIndex(u, s, tx, history, ctx.Done())
					},
				}
			},
		},
		{
			ID: stages.StorageHistoryIndex,
			Build: func(world StageParameters) *Stage {
				return &Stage{
					ID:                  stages.StorageHistoryIndex,
					Description:         "Generate storage history index",
					Disabled:            !sm.History,
					DisabledDescription: "Enable by adding `h` to --storage-mode",
					ExecFunc: func(s *StageState, u Unwinder) error {
						var tx ethdb.RwTx
						if hasTx, ok := world.TX.(ethdb.HasTx); ok {
							tx = hasTx.Tx().(ethdb.RwTx)
						}
						return SpawnStorageHistoryIndex(s, tx, history, ctx.Done())
					},
					UnwindFunc: func(u *UnwindState, s *StageState) error {
						var tx ethdb.RwTx
						if hasTx, ok := world.TX.(ethdb.HasTx); ok {
							tx = hasTx.Tx().(ethdb.RwTx)
						}
						return UnwindStorageHistoryIndex(u, s, tx, history, ctx.Done())
					},
				}
			},
		},
		{
			ID: stages.LogIndex,
			Build: func(world StageParameters) *Stage {
				return &Stage{
					ID:                  stages.LogIndex,
					Description:         "Generate receipt logs index",
					Disabled:            !sm.Receipts,
					DisabledDescription: "Enable by adding `r` to --storage-mode",
					ExecFunc: func(s *StageState, u Unwinder) error {
						var tx ethdb.RwTx
						if hasTx, ok := world.TX.(ethdb.HasTx); ok {
							tx = hasTx.Tx().(ethdb.RwTx)
						}
						return SpawnLogIndex(s, tx, logIndex, ctx.Done())
					},
					UnwindFunc: func(u *UnwindState, s *StageState) error {
						var tx ethdb.RwTx
						if hasTx, ok := world.TX.(ethdb.HasTx); ok {
							tx = hasTx.Tx().(ethdb.RwTx)
						}
						return UnwindLogIndex(u, s, tx, logIndex, ctx.Done())
					},
				}
			},
		},
		{
			ID: stages.CallTraces,
			Build: func(world StageParameters) *Stage {
				return &Stage{
					ID:                  stages.CallTraces,
					Description:         "Generate call traces index",
					DisabledDescription: "Work In Progress",
					Disabled:            !sm.CallTraces,
					ExecFunc: func(s *StageState, u Unwinder) error {
						var tx ethdb.RwTx
						if hasTx, ok := world.TX.(ethdb.HasTx); ok {
							tx = hasTx.Tx().(ethdb.RwTx)
						}
						return SpawnCallTraces(s, tx, ctx.Done(), callTraces)
					},
					UnwindFunc: func(u *UnwindState, s *StageState) error {
						var tx ethdb.RwTx
						if hasTx, ok := world.TX.(ethdb.HasTx); ok {
							tx = hasTx.Tx().(ethdb.RwTx)
						}
						return UnwindCallTraces(u, s, tx, ctx.Done(), callTraces)
					},
				}
			},
		},
		{
			ID: stages.TxLookup,
			Build: func(world StageParameters) *Stage {
				return &Stage{
					ID:                  stages.TxLookup,
					Description:         "Generate tx lookup index",
					Disabled:            !sm.TxIndex,
					DisabledDescription: "Enable by adding `t` to --storage-mode",
					ExecFunc: func(s *StageState, u Unwinder) error {
						var tx ethdb.RwTx
						if hasTx, ok := world.TX.(ethdb.HasTx); ok {
							tx = hasTx.Tx().(ethdb.RwTx)
						}
						return SpawnTxLookup(s, tx, txLookup, ctx.Done())
					},
					UnwindFunc: func(u *UnwindState, s *StageState) error {
						var tx ethdb.RwTx
						if hasTx, ok := world.TX.(ethdb.HasTx); ok {
							tx = hasTx.Tx().(ethdb.RwTx)
						}
						return UnwindTxLookup(u, s, tx, txLookup, ctx.Done())
					},
				}
			},
		},
		{
			ID: stages.TxPool,
			Build: func(world StageParameters) *Stage {
				return &Stage{
					ID:          stages.TxPool,
					Description: "Update transaction pool",
					ExecFunc: func(s *StageState, _ Unwinder) error {
						var tx ethdb.RwTx
						if hasTx, ok := world.TX.(ethdb.HasTx); ok {
							tx = hasTx.Tx().(ethdb.RwTx)
						}
						return SpawnTxPool(s, tx, txPool, ctx.Done())
					},
					UnwindFunc: func(u *UnwindState, s *StageState) error {
						var tx ethdb.RwTx
						if hasTx, ok := world.TX.(ethdb.HasTx); ok {
							tx = hasTx.Tx().(ethdb.RwTx)
						}
						return UnwindTxPool(u, s, tx, txPool, ctx.Done())
					},
				}
			},
		},
		{
			ID: stages.Finish,
			Build: func(world StageParameters) *Stage {
				return &Stage{
					ID:          stages.Finish,
					Description: "Final: update current block for the RPC API",
					ExecFunc: func(s *StageState, _ Unwinder) error {
						var tx ethdb.RwTx
						if hasTx, ok := world.TX.(ethdb.HasTx); ok {
							tx = hasTx.Tx().(ethdb.RwTx)
						}
						return FinishForward(s, world.DB, world.notifier, tx, world.btClient, world.SnapshotBuilder)
					},
					UnwindFunc: func(u *UnwindState, s *StageState) error {
						var tx ethdb.RwTx
						if hasTx, ok := world.TX.(ethdb.HasTx); ok {
							tx = hasTx.Tx().(ethdb.RwTx)
						}
						return UnwindFinish(u, s, world.DB, tx)
					},
				}
			},
		},
	}
}

func ReplacementUnwindOrder() UnwindOrder {
	return []int{
		0, 1, 2, // download headers/bodies
		// Unwinding of tx pool (reinjecting transactions into the pool needs to happen after unwinding execution)
		// also tx pool is before senders because senders unwind is inside cycle transaction
		12,
		3, 4, // senders, exec
		6, 5, // Unwinding of IHashes needs to happen after unwinding HashState
		7, 8, // history
		9,  // log index
		10, // call traces
		11, // tx lookup
	}
}
