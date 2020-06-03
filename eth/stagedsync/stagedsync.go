package stagedsync

import (
	"fmt"

	"github.com/ledgerwatch/turbo-geth/core"
	"github.com/ledgerwatch/turbo-geth/eth/stagedsync/stages"
	"github.com/ledgerwatch/turbo-geth/ethdb"
	"github.com/ledgerwatch/turbo-geth/log"
)

func DoStagedSyncWithFetchers(
	d DownloaderGlue,
	blockchain BlockChain,
	stateDB ethdb.Database,
	pid string,
	history bool,
	datadir string,
	quitCh chan struct{},
	headersFetchers []func() error,
) error {
	var err error
	defer log.Info("Staged sync finished")

	var syncHeadNumber uint64

	stages := []*Stage{
		{
			ID:          stages.Headers,
			Description: "Downloading headers",
			ExecFunc: func(s *StageState) error {
				return DownloadHeaders(s, d, stateDB, headersFetchers, quitCh)
			},
		},
		{
			ID:          stages.Bodies,
			Description: "Downloading block bodiess",
			ExecFunc: func(s *StageState) error {
				cont := true
				for cont && err == nil {
					cont, err = spawnBodyDownloadStage(s, stateDB, d, pid)
					if err != nil {
						return err
					}
				}
				s.Done(stateDB.NewBatch(), 0) // just to proceed to the next stage
				return nil
			},
		},
		{
			ID:          stages.Senders,
			Description: "Recovering senders from tx signatures",
			ExecFunc: func(s *StageState) error {
				return spawnRecoverSendersStage(s, stateDB, blockchain.Config(), quitCh)
			},
		},
		{
			ID:          stages.Execution,
			Description: "Executing blocks w/o hash checks",
			ExecFunc: func(s *StageState) error {
				// TODO: Get rid of a global variable
				syncHeadNumber, err = spawnExecuteBlocksStage(s, stateDB, blockchain, quitCh)
				return err
			},
		},
		{
			ID:          stages.HashCheck,
			Description: "Validating final hashs",
			ExecFunc: func(s *StageState) error {
				return spawnCheckFinalHashStage(s, stateDB, syncHeadNumber, datadir, quitCh)
			},
		},
		{
			ID:          stages.AccountHistoryIndex,
			Description: "Generating account history index",
			ExecFunc: func(s *StageState) error {
				if history {
					return spawnAccountHistoryIndex(s, stateDB, datadir, core.UsePlainStateExecution, quitCh)
				}
				log.Info("Generating account history index is disabled. Enable by adding `h` to --storage-mode")
				return s.Done(stateDB.NewBatch(), 0)
			},
		},
		{
			ID:          stages.StorageHistoryIndex,
			Description: "Generating storage history index",
			ExecFunc: func(s *StageState) error {
				if history {
					return spawnStorageHistoryIndex(s, stateDB, datadir, core.UsePlainStateExecution, quitCh)
				}
				log.Info("Generating storage history index is disabled. Enable by adding `h` to --storage-mode")
				return s.Done(stateDB.NewBatch(), 0)
			},
		},
	}

	state := NewState(stages)

	i := 1

	for !state.IsDone() {
		stage := state.CurrentStage()

		stageState, err := state.StageState(stage.ID, stateDB)
		if err != nil {
			return err
		}

		message := fmt.Sprintf("Sync stage %d/%d. %v...", i, state.Len(), stage.Description)
		log.Info(message)

		err = stage.ExecFunc(stageState)
		if err != nil {
			return err
		}

		log.Info(fmt.Sprintf("%s DONE!", message))

		i++
	}

	return nil
}

func DownloadHeaders(s *StageState, d DownloaderGlue, stateDB ethdb.Database, headersFetchers []func() error, quitCh chan struct{}) error {
	err := d.SpawnSync(headersFetchers)
	if err != nil {
		return err
	}

	log.Info("Checking for unwinding...")
	// Check unwinds backwards and if they are outstanding, invoke corresponding functions
	for stage := stages.Finish - 1; stage > stages.Headers; stage-- {
		unwindPoint, err := stages.GetStageUnwind(stateDB, stage)
		if err != nil {
			return err
		}

		if unwindPoint == 0 {
			continue
		}

		switch stage {
		case stages.Bodies:
			err = unwindBodyDownloadStage(stateDB, unwindPoint)
		case stages.Senders:
			err = unwindSendersStage(stateDB, unwindPoint)
		case stages.Execution:
			err = unwindExecutionStage(unwindPoint, stateDB)
		case stages.HashCheck:
			err = unwindHashCheckStage(unwindPoint, stateDB)
		case stages.AccountHistoryIndex:
			err = unwindAccountHistoryIndex(unwindPoint, stateDB, core.UsePlainStateExecution, quitCh)
		case stages.StorageHistoryIndex:
			err = unwindStorageHistoryIndex(unwindPoint, stateDB, core.UsePlainStateExecution, quitCh)
		default:
			return fmt.Errorf("unrecognized stage for unwinding: %d", stage)
		}

		if err != nil {
			return fmt.Errorf("error unwinding stage: %d: %w", stage, err)
		}
	}
	log.Info("Checking for unwinding... Complete!")

	return s.Done(stateDB.NewBatch(), 0) // just temporary to go to the next step
}
