package voting_tree

import (
	"sync/atomic"

	"github.com/erigontech/erigon-lib/common"
)

const branchingFactor = 128

type entry struct {
	blockRoot   common.Hash
	totalWeight *atomic.Uint64
}

type VotingTree struct {
	l [][]entry
}

func NewVotingTree() *VotingTree {
	return &VotingTree{
		l: make([][]entry, branchingFactor),
	}
}

func (vt *VotingTree) Add(blockRoot common.Hash, weight uint64) {
	idx := blockRoot[0] % branchingFactor
	for i := range vt.l[idx] {
		if vt.l[idx][i].blockRoot == blockRoot {
			vt.l[idx][i].totalWeight.Add(weight)
			return
		}
	}
	newEntry := &atomic.Uint64{}
	newEntry.Store(weight)
	vt.l[idx] = append(vt.l[idx], entry{blockRoot: blockRoot, totalWeight: newEntry})
}

func (vt *VotingTree) Map() map[common.Hash]uint64 {
	res := make(map[common.Hash]uint64)
	for _, entries := range vt.l {
		for _, e := range entries {
			res[e.blockRoot] += e.totalWeight.Load()
		}
	}
	return res
}
