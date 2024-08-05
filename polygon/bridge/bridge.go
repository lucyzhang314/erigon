// Copyright 2024 The Erigon Authors
// This file is part of Erigon.
//
// Erigon is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// Erigon is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with Erigon. If not, see <http://www.gnu.org/licenses/>.

package bridge

import (
	"context"
	"errors"
	"fmt"
	"sync/atomic"
	"time"

	"github.com/erigontech/erigon-lib/kv"
	"github.com/erigontech/erigon-lib/log/v3"
	"github.com/erigontech/erigon/common/u256"
	"github.com/erigontech/erigon/core"
	"github.com/erigontech/erigon/core/state"
	bortypes "github.com/erigontech/erigon/polygon/bor/types"
	"github.com/erigontech/erigon/polygon/polygoncommon"

	libcommon "github.com/erigontech/erigon-lib/common"
	"github.com/erigontech/erigon/accounts/abi"
	"github.com/erigontech/erigon/core/types"
	"github.com/erigontech/erigon/polygon/bor/borcfg"
	"github.com/erigontech/erigon/polygon/heimdall"
)

var ErrMapNotAvailable = errors.New("map not available")
var ErrTxNotAvailable = errors.New("tx not available")

type fetchSyncEventsType func(ctx context.Context, fromId uint64, to time.Time, limit int) ([]*heimdall.EventRecordWithTime, error)

type Bridge struct {
	store                    Store
	ready                    atomic.Bool
	lastProcessedBlockNumber atomic.Uint64
	lastProcessedEventID     atomic.Uint64

	log                log.Logger
	borConfig          *borcfg.BorConfig
	stateReceiverABI   abi.ABI
	stateClientAddress libcommon.Address
	fetchSyncEvents    fetchSyncEventsType
}

func Assemble(dataDir string, logger log.Logger, borConfig *borcfg.BorConfig, fetchSyncEvents fetchSyncEventsType, stateReceiverABI abi.ABI) *Bridge {
	bridgeDB := polygoncommon.NewDatabase(dataDir, kv.PolygonBridgeDB, databaseTablesCfg, logger)
	bridgeStore := NewStore(bridgeDB)
	return NewBridge(bridgeStore, logger, borConfig, fetchSyncEvents, stateReceiverABI)
}

func NewBridge(store Store, logger log.Logger, borConfig *borcfg.BorConfig, fetchSyncEvents fetchSyncEventsType, stateReceiverABI abi.ABI) *Bridge {
	return &Bridge{
		store:              store,
		log:                logger,
		borConfig:          borConfig,
		fetchSyncEvents:    fetchSyncEvents,
		stateReceiverABI:   stateReceiverABI,
		stateClientAddress: libcommon.HexToAddress(borConfig.StateReceiverContract),
	}
}

func (b *Bridge) Run(ctx context.Context) error {
	err := b.store.Prepare(ctx)
	if err != nil {
		return err
	}
	defer b.Close()

	// get last known sync ID
	lastEventID, err := b.store.GetLatestEventID(ctx)
	if err != nil {
		return err
	}

	lastProcessedEventID, err := b.store.GetLastProcessedEventID(ctx)
	if err != nil {
		return err
	}

	b.lastProcessedEventID.Store(lastProcessedEventID)

	// start syncing
	b.log.Debug(bridgeLogPrefix("Bridge is running"), "lastEventID", lastEventID)

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		// get all events from last sync ID to now
		to := time.Now()
		events, err := b.fetchSyncEvents(ctx, lastEventID+1, to, 0)
		if err != nil {
			return err
		}

		if len(events) != 0 {
			b.ready.Store(false)
			if err := b.store.AddEvents(ctx, events, b.stateReceiverABI); err != nil {
				return err
			}

			lastEventID = events[len(events)-1].ID
		} else {
			b.ready.Store(true)
			if err := libcommon.Sleep(ctx, 30*time.Second); err != nil {
				return err
			}
		}

		b.log.Debug(bridgeLogPrefix(fmt.Sprintf("got %v new events, last event ID: %v, ready: %v", len(events), lastEventID, b.ready.Load())))
	}
}

func (b *Bridge) Close() {
	b.store.Close()
}

// EngineService interface implementations

// ProcessNewBlocks iterates through all blocks and constructs a map from block number to sync events
func (b *Bridge) ProcessNewBlocks(ctx context.Context, blocks []*types.Block) error {
	eventMap := make(map[libcommon.Hash]uint64)
	txMap := make(map[libcommon.Hash]uint64)
	var prevSprintTime time.Time

	for _, block := range blocks {
		// check if block is start of span
		if !b.isSprintStart(block.NumberU64()) {
			continue
		}

		var timeLimit time.Time
		if b.borConfig.IsIndore(block.NumberU64()) {
			stateSyncDelay := b.borConfig.CalculateStateSyncDelay(block.NumberU64())
			timeLimit = time.Unix(int64(block.Time()-stateSyncDelay), 0)
		} else {
			timeLimit = prevSprintTime
		}

		prevSprintTime = time.Unix(int64(block.Time()), 0)

		lastDBID, err := b.store.GetSprintLastEventID(ctx, b.lastProcessedEventID.Load(), timeLimit, b.stateReceiverABI)
		if err != nil {
			return err
		}

		if lastDBID > b.lastProcessedEventID.Load() {
			b.log.Debug(bridgeLogPrefix(fmt.Sprintf("Creating map for block %d, start ID %d, end ID %d", block.NumberU64(), b.lastProcessedEventID.Load(), lastDBID)))

			k := bortypes.ComputeBorTxHash(block.NumberU64(), block.Hash())
			eventMap[k] = b.lastProcessedEventID.Load()
			txMap[k] = block.NumberU64()

			b.lastProcessedEventID.Store(lastDBID)
		}

		b.lastProcessedBlockNumber.Store(block.NumberU64())
	}

	err := b.store.StoreEventID(ctx, eventMap)
	if err != nil {
		return err
	}

	err = b.store.StoreTxMap(ctx, txMap)
	if err != nil {
		return err
	}

	return nil
}

// Synchronize blocks till bridge has map at tip
func (b *Bridge) Synchronize(ctx context.Context, tip *types.Header) error {
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
		if b.ready.Load() || b.lastProcessedBlockNumber.Load() >= tip.Number.Uint64() {
			return nil
		}
	}
}

// Unwind deletes map entries till tip
func (b *Bridge) Unwind(ctx context.Context, tip *types.Header) error {
	k := bortypes.ComputeBorTxHash(tip.Number.Uint64(), tip.Hash())
	return b.store.PruneEventIDs(ctx, k)
}

// Events returns all sync events at blockNum
func (b *Bridge) Events(ctx context.Context, borTxHash libcommon.Hash) ([]*types.Message, error) {
	start, end, err := b.store.GetEventIDRange(ctx, borTxHash)
	if err != nil {
		return nil, err
	}

	if end == 0 { // exception for tip processing
		end = b.lastProcessedEventID.Load()
	}

	eventsRaw := make([]*types.Message, 0, end-start+1)

	// get events from DB
	events, err := b.store.GetEvents(ctx, start+1, end+1)
	if err != nil {
		return nil, err
	}

	b.log.Debug(bridgeLogPrefix(fmt.Sprintf("got %v events for tx %v", len(events), borTxHash)))

	// convert to message
	for _, event := range events {
		msg := types.NewMessage(
			state.SystemAddress,
			&b.stateClientAddress,
			0, u256.Num0,
			core.SysCallGasLimit,
			u256.Num0,
			nil, nil,
			event, nil, false,
			true,
			nil,
		)

		eventsRaw = append(eventsRaw, &msg)
	}

	return eventsRaw, nil
}

func (b *Bridge) TxLookup(ctx context.Context, borTxHash libcommon.Hash) (uint64, error) {
	blockNum, err := b.store.TxMap(ctx, borTxHash)
	if err != nil {
		return 0, err
	}

	return blockNum, nil
}

// Helper functions
func (b *Bridge) isSprintStart(headerNum uint64) bool {
	if headerNum%b.borConfig.CalculateSprintLength(headerNum) != 0 || headerNum == 0 {
		return false
	}

	return true
}
