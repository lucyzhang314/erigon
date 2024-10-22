package voting_tree_test

import (
	"testing"

	"github.com/erigontech/erigon-lib/common"
	"github.com/erigontech/erigon/cl/utils/voting_tree"
	"github.com/stretchr/testify/require"
)

func TestVotingTreeSim(t *testing.T) {
	vt := voting_tree.NewVotingTree()
	vt.Add(common.Hash{0x01}, 1)
	vt.Add(common.Hash{0x02}, 2)
	vt.Add(common.Hash{0x03}, 3)

	m := vt.Map()

	if len(m) != 3 {
		t.Errorf("Expected 3 entries, got %d", len(m))
	}

	if m[common.Hash{0x01}] != 1 {
		t.Errorf("Expected 1, got %d", m[common.Hash{0x01}])
	}

	if m[common.Hash{0x02}] != 2 {
		t.Errorf("Expected 2, got %d", m[common.Hash{0x02}])
	}

	if m[common.Hash{0x03}] != 3 {
		t.Errorf("Expected 3, got %d", m[common.Hash{0x03}])
	}
}

func TestVotingTreeSimParallel(t *testing.T) {
	vtParallel := voting_tree.NewVotingTree()

	// Add 1000 entries in parallel
	for i := 0; i < 10000; i++ {
		go vtParallel.Add(common.Hash{byte(i)}, uint64(1))
	}

	vtSequential := voting_tree.NewVotingTree()

	// Add 1000 entries sequentially
	for i := 0; i < 10000; i++ {
		vtSequential.Add(common.Hash{byte(i)}, uint64(1))
	}

	mParallel := vtParallel.Map()
	mSequential := vtSequential.Map()

	require.Equal(t, len(mParallel), len(mSequential))
	for k, v := range mParallel {
		require.Equal(t, v, mSequential[k])
	}
}
