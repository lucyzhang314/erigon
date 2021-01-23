package bodydownload

import (
	//"context"
	//"github.com/ledgerwatch/turbo-geth/common/dbutils"

	"fmt"

	"github.com/ledgerwatch/turbo-geth/common"
	"github.com/ledgerwatch/turbo-geth/core/rawdb"
	"github.com/ledgerwatch/turbo-geth/core/types"
	"github.com/ledgerwatch/turbo-geth/eth"
	"github.com/ledgerwatch/turbo-geth/eth/stagedsync/stages"
	"github.com/ledgerwatch/turbo-geth/ethdb"
	"github.com/ledgerwatch/turbo-geth/log"
)

const BlockBufferSize = 128

// UpdateFromDb reads the state of the database and refreshes the state of the body download
func (bd *BodyDownload) UpdateFromDb(db ethdb.Database) error {
	var headerProgress, bodyProgress uint64
	var err error
	headerProgress, err = stages.GetStageProgress(db, stages.Headers)
	if err != nil {
		return err
	}
	bd.maxProgress = headerProgress + 1
	bodyProgress, err = stages.GetStageProgress(db, stages.Bodies)
	if err != nil {
		return err
	}
	bd.lock.Lock()
	defer bd.lock.Unlock()
	// Resetting for requesting a new range of blocks
	bd.requestedLow = bodyProgress + 1
	bd.lowWaitUntil = 0
	bd.requestHigh = bd.requestedLow + (bd.outstandingLimit / 2)
	bd.requestedMap = make(map[DoubleHash]uint64)
	bd.delivered.Clear()
	for i := 0; i < len(bd.deliveries); i++ {
		bd.deliveries[i] = nil
	}
	return nil
}

func (bd *BodyDownload) RequestMoreBodies(db ethdb.Database, blockNum uint64, currentTime, timeWithTimeout uint64) (*BodyRequest, uint64) {
	bd.lock.Lock()
	defer bd.lock.Unlock()
	if blockNum < bd.requestedLow && currentTime >= bd.lowWaitUntil {
		blockNum = bd.requestedLow
		bd.lowWaitUntil = timeWithTimeout
	} else {
		return nil, blockNum
	}
	var bodyReq *BodyRequest
	blockNums := make([]uint64, 0, BlockBufferSize)
	hashes := make([]common.Hash, 0, BlockBufferSize)
	for ; len(blockNums) < BlockBufferSize && blockNum < bd.maxProgress; blockNum++ {
		// Check if we reached highest allowed request block number, and turn back
		if blockNum >= bd.requestHigh {
			bd.requestHigh = bd.requestedLow + bd.outstandingLimit
			//bd.feedDeliveries()
			blockNum = 0
			break // Avoid tight loop
		}
		if bd.delivered.Contains(blockNum) {
			// Already delivered, no need to request
			continue
		}
		var hash common.Hash
		var header *types.Header
		var err error
		if bd.deliveries[blockNum-bd.requestedLow] != nil {
			// If this block was requested before, we don't need to fetch the headers from the database the second time
			header = bd.deliveries[blockNum-bd.requestedLow].Header()
			hash = header.Hash()
		} else {
			hash, err = rawdb.ReadCanonicalHash(db, blockNum)
			if err == nil {
				header = rawdb.ReadHeader(db, hash, blockNum)
			} else {
				log.Error("Could not find canonical header", "block number", blockNum)
			}
			if header != nil {
				bd.deliveries[blockNum-bd.requestedLow] = types.NewBlockWithHeader(header) // Block without uncles and transactions
				if header.UncleHash != types.EmptyUncleHash || header.TxHash != types.EmptyRootHash {
					var doubleHash DoubleHash
					copy(doubleHash[:], header.UncleHash.Bytes())
					copy(doubleHash[common.HashLength:], header.TxHash.Bytes())
					bd.requestedMap[doubleHash] = blockNum
				}
			}
		}
		if header == nil {
			log.Error("Header not found", "block number", blockNum)
			panic("")
		} else if header.UncleHash != types.EmptyUncleHash || header.TxHash != types.EmptyRootHash {
			blockNums = append(blockNums, blockNum)
			hashes = append(hashes, hash)
		} else {
			// Both uncleHash and txHash are empty, no need to request
			bd.delivered.Add(blockNum)
		}
	}
	if len(blockNums) > 0 {
		bodyReq = &BodyRequest{BlockNums: blockNums, Hashes: hashes}
		fmt.Printf("Generated request for %d bodies [%d-%d]\n", len(blockNums), blockNums[0], blockNums[len(blockNums)-1])
	}
	return bodyReq, blockNum
}

// DeliverBody takes the block body received from a peer and adds it to the various data structures
func (bd *BodyDownload) DeliverBody(body *eth.BlockBody) (uint64, bool) {
	uncleHash := types.CalcUncleHash(body.Uncles)
	txHash := types.DeriveSha(types.Transactions(body.Transactions))
	var doubleHash DoubleHash
	copy(doubleHash[:], uncleHash.Bytes())
	copy(doubleHash[common.HashLength:], txHash.Bytes())
	bd.lock.Lock()
	defer bd.lock.Unlock()
	// Block numbers are added to the bd.delivered bitmap here, only for blocks for which the body has been received, and their double hashes are present in the bd.requesredMap
	// Also, block numbers can be added to bd.delivered for empty blocks, above
	if blockNum, ok := bd.requestedMap[doubleHash]; ok {
		bd.delivered.Add(blockNum)
		bd.deliveries[blockNum-bd.requestedLow] = bd.deliveries[blockNum-bd.requestedLow].WithBody(body.Transactions, body.Uncles)
		delete(bd.requestedMap, doubleHash) // Delivered, cleaning up
		return blockNum, true
	}
	return 0, false
}

func (bd *BodyDownload) GetDeliveries() []*types.Block {
	bd.lock.Lock()
	defer bd.lock.Unlock()
	var i uint64
	for i = 0; !bd.delivered.IsEmpty() && bd.requestedLow+i == bd.delivered.Minimum(); i++ {
		bd.delivered.Remove(bd.requestedLow + i)
	}
	// Move the deliveries back
	// bd.requestedLow can only be moved forward if there are consequitive block numbers present in the bd.delivered map
	var d []*types.Block
	if i > 0 {
		d = make([]*types.Block, i)
		copy(d, bd.deliveries[:i])
		copy(bd.deliveries[:], bd.deliveries[i:])
		for j := len(bd.deliveries) - int(i); j < len(bd.deliveries); j++ {
			bd.deliveries[j] = nil
		}
		bd.requestedLow += i
	}
	return d
}

func (bd *BodyDownload) feedDeliveries() {
	var i uint64
	var channelFull bool
	for i = 0; !channelFull && !bd.delivered.IsEmpty() && bd.requestedLow+i == bd.delivered.Minimum(); i++ {
		select {
		case bd.blockChannel <- bd.deliveries[i]:
			// This is delivery
			bd.delivered.Remove(bd.requestedLow + i)
		default:
			// Delivery is not possible for now
			channelFull = true
		}
	}
	// Move the deliveries back
	// bd.requestedLow can only be moved forward if there are consequitive block numbers present in the bd.delivered map
	if i > 0 {
		copy(bd.deliveries[:], bd.deliveries[i:])
		for j := len(bd.deliveries) - int(i); j < len(bd.deliveries); j++ {
			bd.deliveries[j] = nil
		}
		bd.requestedLow += i
	}
}

func (bd *BodyDownload) FeedDeliveries() {
	bd.lock.Lock()
	defer bd.lock.Unlock()
	bd.feedDeliveries()
}

func (bd *BodyDownload) CloseStageData() {
	bd.lock.Lock()
	defer bd.lock.Unlock()
	if bd.blockChannel != nil {
		close(bd.blockChannel)
	}
}

func (bd *BodyDownload) PrepareStageData() chan *types.Block {
	return bd.blockChannel
}
