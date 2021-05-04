package changeset

import (
	"bytes"
	"sort"

	"github.com/ledgerwatch/turbo-geth/common"
	"github.com/ledgerwatch/turbo-geth/common/dbutils"
	"github.com/ledgerwatch/turbo-geth/ethdb"
)

type Encoder func(blockN uint64, s *ChangeSet, f func(k, v []byte) error) error
type Decoder func(dbKey, dbValue []byte) (blockN uint64, k, v []byte)

func NewAccountChangeSet() *ChangeSet {
	return &ChangeSet{
		Changes: make([]Change, 0),
		keyLen:  common.AddressLength,
	}
}

func EncodeAccounts(blockN uint64, s *ChangeSet, f func(k, v []byte) error) error {
	sort.Sort(s)
	newK := dbutils.EncodeBlockNumber(blockN)
	for _, cs := range s.Changes {
		newV := make([]byte, len(cs.Key)+len(cs.Value))
		copy(newV, cs.Key)
		copy(newV[len(cs.Key):], cs.Value)
		if err := f(newK, newV); err != nil {
			return err
		}
	}
	return nil
}

type AccountChangeSet struct{ c ethdb.CursorDupSort }

func (b AccountChangeSet) Find(blockNumber uint64, key []byte) ([]byte, error) {
	fromDBFormat := FromDBFormat(common.AddressLength)
	k := dbutils.EncodeBlockNumber(blockNumber)
	v, err := b.c.SeekBothRange(k, key)
	if err != nil {
		return nil, err
	}
	_, k, v = fromDBFormat(k, v)
	if !bytes.HasPrefix(k, key) {
		return nil, nil
	}
	return v, nil
}

// GetModifiedAccounts returns a list of addresses that were modified in the block range
func GetModifiedAccounts(db ethdb.Tx, startNum, endNum uint64) ([]common.Address, error) {
	changedAddrs := make(map[common.Address]struct{})
	if err := Walk(db, dbutils.PlainAccountChangeSetBucket, dbutils.EncodeBlockNumber(startNum), 0, func(blockN uint64, k, v []byte) (bool, error) {
		if blockN > endNum {
			return false, nil
		}
		changedAddrs[common.BytesToAddress(k)] = struct{}{}
		return true, nil
	}); err != nil {
		return nil, err
	}

	if len(changedAddrs) == 0 {
		return nil, nil
	}

	idx := 0
	result := make([]common.Address, len(changedAddrs))
	for addr := range changedAddrs {
		copy(result[idx][:], addr[:])
		idx++
	}

	return result, nil
}
