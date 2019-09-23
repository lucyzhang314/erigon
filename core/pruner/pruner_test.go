package pruner

import (
	"bytes"
	"crypto/ecdsa"
	"github.com/davecgh/go-spew/spew"
	"github.com/ledgerwatch/turbo-geth/accounts/abi/bind"
	"github.com/ledgerwatch/turbo-geth/accounts/abi/bind/backends"
	"github.com/ledgerwatch/turbo-geth/common"
	"github.com/ledgerwatch/turbo-geth/common/dbutils"
	"github.com/ledgerwatch/turbo-geth/consensus/ethash"
	"github.com/ledgerwatch/turbo-geth/core"
	"github.com/ledgerwatch/turbo-geth/core/types"
	"github.com/ledgerwatch/turbo-geth/core/types/accounts"
	"github.com/ledgerwatch/turbo-geth/core/vm"
	"github.com/ledgerwatch/turbo-geth/crypto"
	"github.com/ledgerwatch/turbo-geth/ethdb"
	"github.com/ledgerwatch/turbo-geth/params"
	"github.com/ledgerwatch/turbo-geth/tests/contracts"
	"math/big"
	"reflect"
	"testing"
)

func TestBasisAccountPruning(t *testing.T) {
	// Configure and generate a sample block chain
	var (
		db       = ethdb.NewMemDatabase()
		key, _   = crypto.HexToECDSA("b71c71a67e1177ad4e901695e1b4b9ee17ae16c6668d313eac2f96dbcda3f291")
		key1, _  = crypto.HexToECDSA("49a7b37aa6f6645917e7b807e9d1c00d4fa71f18343b0d4122a4d2df64dd6fee")
		key2, _  = crypto.HexToECDSA("8a1f9a8f95be41cd7ccb6168179afb4504aefe388d1e14474d32c45c72ce7b7a")
		address  = crypto.PubkeyToAddress(key.PublicKey)
		address1 = crypto.PubkeyToAddress(key1.PublicKey)
		address2 = crypto.PubkeyToAddress(key2.PublicKey)
		theAddr  = common.Address{1}
		funds    = big.NewInt(1000000000)
		gspec    = &core.Genesis{
			Config: &params.ChainConfig{
				ChainID:             big.NewInt(1),
				HomesteadBlock:      new(big.Int),
				EIP155Block:         new(big.Int),
				EIP158Block:         big.NewInt(1),
				EIP2027Block:        big.NewInt(4),
				ConstantinopleBlock: big.NewInt(1),
			},
			Alloc: core.GenesisAlloc{
				address:  {Balance: funds},
				address1: {Balance: funds},
				address2: {Balance: funds},
			},
		}
		genesis   = gspec.MustCommit(db)
		genesisDb = db.MemCopy()
		// this code generates a log
		signer = types.HomesteadSigner{}
	)

	numBlocks := 10
	engine := ethash.NewFaker()
	blockchain, err := core.NewBlockChain(db, nil, gspec.Config, engine, vm.Config{}, nil)
	if err != nil {
		t.Fatal(err)
	}

	blocks, _ := core.GenerateChain(gspec.Config, genesis, engine, genesisDb, numBlocks, func(i int, block *core.BlockGen) {
		var (
			tx     *types.Transaction
			genErr error
		)
		var addr common.Address
		var k *ecdsa.PrivateKey
		switch i % 3 {
		case 0:
			addr = address
			k = key
		case 1:
			addr = address1
			k = key1
		case 2:
			addr = address2
			k = key2
		}
		tx, genErr = types.SignTx(types.NewTransaction(block.TxNonce(addr), theAddr, big.NewInt(1000), 21000, new(big.Int), nil), signer, k)
		if genErr != nil {
			t.Fatal(genErr)
		}
		block.AddTx(tx)
	})

	for i := range blocks {
		_, err = blockchain.InsertChain(types.Blocks{blocks[i]})
		if err != nil {
			t.Fatal(err)
		}
	}

	res, err := getStat(db)
	if err != nil {
		t.Fatal(err)
	}
	expected := stateStats{
		NotFoundAccountsInHistory:     5,
		ErrAccountsInHistory:          0,
		ErrDecodedAccountsInHistory:   0,
		NumOfChangesInAccountsHistory: 28,
		AccountSuffixRecordsByTimestamp: map[uint64]uint32{
			0:  3,
			1:  3,
			2:  3,
			3:  3,
			4:  3,
			5:  3,
			6:  3,
			7:  3,
			8:  3,
			9:  3,
			10: 3,
		},
		StorageSuffixRecordsByTimestamp: map[uint64]uint32{},
		AccountsInState:                 5,
	}
	if !reflect.DeepEqual(expected, res) {
		spew.Dump(res)
		t.Fatal("Not equal")
	}

	err = Prune(db, 0, uint64(numBlocks)-1)
	if err != nil {
		t.Fatal(err)
	}

	res, err = getStat(db)
	if err != nil {
		t.Fatal(err)
	}
	expected = stateStats{
		NotFoundAccountsInHistory:     0,
		ErrAccountsInHistory:          0,
		ErrDecodedAccountsInHistory:   0,
		NumOfChangesInAccountsHistory: 3,
		AccountSuffixRecordsByTimestamp: map[uint64]uint32{
			10: 3,
		},
		StorageSuffixRecordsByTimestamp: map[uint64]uint32{},
		AccountsInState:                 5,
	}
	if !reflect.DeepEqual(expected, res) {
		spew.Dump(res)
		t.Fatal("Not equal")
	}

	err = Prune(db, uint64(numBlocks)-1, uint64(numBlocks))
	if err != nil {
		t.Fatal(err)
	}
	res, err = getStat(db)
	if err != nil {
		t.Fatal(err)
	}
	expected = stateStats{
		NotFoundAccountsInHistory:       0,
		ErrAccountsInHistory:            0,
		ErrDecodedAccountsInHistory:     0,
		NumOfChangesInAccountsHistory:   0,
		AccountSuffixRecordsByTimestamp: map[uint64]uint32{},
		StorageSuffixRecordsByTimestamp: map[uint64]uint32{},
		AccountsInState:                 5,
	}
	if !reflect.DeepEqual(expected, res) {
		spew.Dump(res)
		t.Fatal("Not equal")
	}

}

type stateStats struct {
	NotFoundAccountsInHistory       uint64
	ErrAccountsInHistory            uint64
	ErrDecodedAccountsInHistory     uint64
	NumOfChangesInAccountsHistory   uint64
	AccountSuffixRecordsByTimestamp map[uint64]uint32
	StorageSuffixRecordsByTimestamp map[uint64]uint32
	AccountsInState                 uint64
}

func getStat(db *ethdb.BoltDatabase) (stateStats, error) {
	stat := stateStats{
		AccountSuffixRecordsByTimestamp: make(map[uint64]uint32, 0),
		StorageSuffixRecordsByTimestamp: make(map[uint64]uint32, 0),
	}
	err := db.Walk(dbutils.SuffixBucket, []byte{}, 0, func(key, v []byte) (b bool, e error) {
		timestamp, _ := dbutils.DecodeTimestamp(key)

		changedAccounts := dbutils.Suffix(v)
		if bytes.HasSuffix(key, dbutils.AccountsHistoryBucket) {
			if _, ok := stat.AccountSuffixRecordsByTimestamp[timestamp]; ok {
				panic("multiple account suffix records")
			}
			stat.AccountSuffixRecordsByTimestamp[timestamp] = changedAccounts.KeyCount()
		}
		if bytes.HasSuffix(key, dbutils.StorageHistoryBucket) {
			if _, ok := stat.StorageSuffixRecordsByTimestamp[timestamp]; ok {
				panic("multiple storage suffix records")
			}
			stat.StorageSuffixRecordsByTimestamp[timestamp] = changedAccounts.KeyCount()
		}

		if bytes.HasSuffix(key, dbutils.AccountsHistoryBucket) {
			err := changedAccounts.Walk(func(k []byte) error {
				compKey, _ := dbutils.CompositeKeySuffix(k, timestamp)
				b, err := db.Get(dbutils.AccountsHistoryBucket, compKey)
				if len(b) == 0 {
					stat.NotFoundAccountsInHistory++
					return nil
				}
				if err != nil {
					stat.ErrAccountsInHistory++
				} else {
					acc := &accounts.Account{}
					errInn := acc.DecodeForStorage(b)
					if errInn != nil {
						stat.ErrDecodedAccountsInHistory++
					} else {
						stat.NumOfChangesInAccountsHistory++
					}
				}
				return nil
			})
			if err != nil {
				return false, err
			}
		}

		return true, nil
	})

	if err != nil {
		return stateStats{}, err
	}

	err = db.Walk(dbutils.AccountsBucket, []byte{}, 0, func(key, v []byte) (b bool, e error) {
		stat.AccountsInState++
		return true, nil
	})
	if err != nil {
		return stateStats{}, err
	}
	return stat, nil
}

func TestStoragePruning(t *testing.T) {
	// Configure and generate a sample block chain
	var (
		db       = ethdb.NewMemDatabase()
		key, _   = crypto.HexToECDSA("b71c71a67e1177ad4e901695e1b4b9ee17ae16c6668d313eac2f96dbcda3f291")
		key1, _  = crypto.HexToECDSA("49a7b37aa6f6645917e7b807e9d1c00d4fa71f18343b0d4122a4d2df64dd6fee")
		key2, _  = crypto.HexToECDSA("8a1f9a8f95be41cd7ccb6168179afb4504aefe388d1e14474d32c45c72ce7b7a")
		address  = crypto.PubkeyToAddress(key.PublicKey)
		address1 = crypto.PubkeyToAddress(key1.PublicKey)
		address2 = crypto.PubkeyToAddress(key2.PublicKey)
		funds    = big.NewInt(1000000000)
		gspec    = &core.Genesis{
			Config: &params.ChainConfig{
				ChainID:             big.NewInt(1),
				HomesteadBlock:      new(big.Int),
				EIP155Block:         new(big.Int),
				EIP158Block:         big.NewInt(1),
				EIP2027Block:        big.NewInt(4),
				ConstantinopleBlock: big.NewInt(1),
			},
			Alloc: core.GenesisAlloc{
				address:  {Balance: funds},
				address1: {Balance: funds},
				address2: {Balance: funds},
			},
		}
		genesis   = gspec.MustCommit(db)
		genesisDb = db.MemCopy()
	)

	engine := ethash.NewFaker()
	blockchain, err := core.NewBlockChain(db, nil, gspec.Config, engine, vm.Config{}, nil)
	if err != nil {
		t.Fatal(err)
	}

	blockchain.EnableReceipts(true)

	contractBackend := backends.NewSimulatedBackendWithConfig(gspec.Alloc, gspec.Config, gspec.GasLimit)
	transactOpts := bind.NewKeyedTransactor(key)
	transactOpts1 := bind.NewKeyedTransactor(key1)
	transactOpts2 := bind.NewKeyedTransactor(key2)

	var eipContract *contracts.Eip2027

	blocks, _ := core.GenerateChain(gspec.Config, genesis, engine, genesisDb, 6, func(i int, block *core.BlockGen) {
		var (
			tx       *types.Transaction
			innerErr error
		)

		switch i {
		case 0:
			_, tx, eipContract, innerErr = contracts.DeployEip2027(transactOpts, contractBackend)
			assertNil(t, innerErr)
			block.AddTx(tx)

		case 1:
			tx, innerErr = eipContract.Create(transactOpts1, big.NewInt(1))
			assertNil(t, innerErr)
			block.AddTx(tx)

			tx, innerErr = eipContract.Create(transactOpts2, big.NewInt(2))
			assertNil(t, innerErr)
			block.AddTx(tx)

			tx, innerErr = eipContract.Create(transactOpts, big.NewInt(3))
			assertNil(t, innerErr)
			block.AddTx(tx)
		case 2:
			tx, innerErr = eipContract.Update(transactOpts1, big.NewInt(0))
			assertNil(t, innerErr)
			block.AddTx(tx)

			tx, innerErr = eipContract.Update(transactOpts2, big.NewInt(0))
			assertNil(t, innerErr)
			block.AddTx(tx)

			tx, innerErr = eipContract.Update(transactOpts, big.NewInt(0))
			assertNil(t, innerErr)
			block.AddTx(tx)

		case 3:
			tx, innerErr = eipContract.Update(transactOpts1, big.NewInt(7))
			assertNil(t, innerErr)
			block.AddTx(tx)

			tx, innerErr = eipContract.Update(transactOpts2, big.NewInt(7))
			assertNil(t, innerErr)
			block.AddTx(tx)

			tx, innerErr = eipContract.Update(transactOpts, big.NewInt(7))
			assertNil(t, innerErr)
			block.AddTx(tx)

		case 4:
			tx, innerErr = eipContract.Update(transactOpts1, big.NewInt(5))
			assertNil(t, innerErr)
			block.AddTx(tx)

			tx, innerErr = eipContract.Update(transactOpts2, big.NewInt(5))
			assertNil(t, innerErr)
			block.AddTx(tx)

			tx, innerErr = eipContract.Update(transactOpts, big.NewInt(5))
			assertNil(t, innerErr)
			block.AddTx(tx)

		case 5:
			tx, innerErr = eipContract.Remove(transactOpts1)
			assertNil(t, innerErr)
			block.AddTx(tx)

			tx, innerErr = eipContract.Remove(transactOpts2)
			assertNil(t, innerErr)
			block.AddTx(tx)

			tx, innerErr = eipContract.Remove(transactOpts)
			assertNil(t, innerErr)
			block.AddTx(tx)

		}

		if err != nil {
			t.Fatal(innerErr)
		}

		contractBackend.Commit()
	})

	for i := range blocks {
		_, err = blockchain.InsertChain(types.Blocks{blocks[i]})
		if err != nil {
			t.Fatal(err)
		}
	}

	res, err := getStat(db)
	assertNil(t, err)

	expected := stateStats{
		NotFoundAccountsInHistory:     5,
		ErrAccountsInHistory:          0,
		ErrDecodedAccountsInHistory:   0,
		NumOfChangesInAccountsHistory: 26,
		AccountSuffixRecordsByTimestamp: map[uint64]uint32{
			0: 3,
			1: 3,
			2: 5,
			3: 5,
			4: 5,
			5: 5,
			6: 5,
		},
		StorageSuffixRecordsByTimestamp: map[uint64]uint32{
			1: 1,
			2: 3,
			3: 3,
			4: 3,
			5: 3,
			6: 3,
		},
		AccountsInState: 5,
	}
	if !reflect.DeepEqual(expected, res) {
		spew.Dump(getStat(db))
		t.Fatal("not equals")
	}

	err = Prune(db, 0, 5)
	if err != nil {
		t.Fatal(err)
	}

	spew.Dump(getStat(db))
}

func assertNil(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatal(err)
	}
}
