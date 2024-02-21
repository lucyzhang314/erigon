package p2p

import "time"

const missingBlockNumExpiry = time.Hour

type peerSyncProgress struct {
	peerId               PeerId
	minMissingBlockNum   uint64
	minMissingBlockNumTs time.Time
}

func (psp *peerSyncProgress) blockNumPresent(blockNum uint64) {
	if psp.minMissingBlockNum <= blockNum {
		psp.minMissingBlockNum = 0
		psp.minMissingBlockNumTs = time.Unix(0, 0)
	}
}

func (psp *peerSyncProgress) blockNumMissing(blockNum uint64) {
	if psp.minMissingBlockNum >= blockNum || psp.minMissingBlockNumTsExpired() {
		psp.minMissingBlockNum = blockNum
		psp.minMissingBlockNumTs = time.Now()
	}
}

func (psp *peerSyncProgress) peerMayHaveBlockNum(blockNum uint64) bool {
	if psp.minMissingBlockNumTsExpired() {
		return true
	}

	return psp.minMissingBlockNum <= blockNum
}

func (psp *peerSyncProgress) minMissingBlockNumTsExpired() bool {
	return time.Now().After(psp.minMissingBlockNumTs.Add(missingBlockNumExpiry))
}
