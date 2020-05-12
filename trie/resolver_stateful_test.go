package trie

import (
	"fmt"
	"math/big"
	"testing"

	"github.com/ledgerwatch/turbo-geth/common"
	"github.com/ledgerwatch/turbo-geth/common/dbutils"
	"github.com/ledgerwatch/turbo-geth/core/types/accounts"
	"github.com/ledgerwatch/turbo-geth/crypto"
	"github.com/ledgerwatch/turbo-geth/ethdb"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Put 1 embedded entry into the database and try to resolve it
func TestResolve1(t *testing.T) {
	t.Skip("weird case of abandoned storage, will handle it later")

	require, assert, db := require.New(t), assert.New(t), ethdb.NewMemDatabase()
	putStorage := func(k string, v string) {
		err := db.Put(dbutils.CurrentStateBucket, common.Hex2Bytes(k), common.Hex2Bytes(v))
		require.NoError(err)
	}
	putStorage("aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", "")
	r := NewResolver(0)
	rs := NewResolveSet(0)
	rs.AddKey(common.Hex2Bytes("aaaaabbbbbaaaaabbbbbaaaaabbbbbaa"))
	subTries, err := r.ResolveWithDb(db, 0, rs, [][]byte{common.Hex2Bytes("aaaaabbbbb")}, []int{40}, false)
	require.NoError(err)
	tr := New(common.Hash{})
	tr.HookSubTries(subTries, [][]byte{nil}) // hook up to the root of the trie
	x, ok := tr.Get(common.Hex2Bytes("aaaaabbbbbaaaaabbbbbaa"))
	assert.True(ok)
	assert.NotNil(x)
}

func TestResolve2(t *testing.T) {
	t.Skip("weird case of abandoned storage, will handle it later")

	require, assert, db := require.New(t), assert.New(t), ethdb.NewMemDatabase()
	putStorage := func(k string, v string) {
		err := db.Put(dbutils.CurrentStateBucket, common.Hex2Bytes(k), common.Hex2Bytes(v))
		require.NoError(err)
	}
	putStorage("aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", "")
	putStorage("aaaaaccccccccccccccccccccccccccc", "")

	r := NewResolver(0)
	rs := NewResolveSet(0)
	rs.AddKey(common.Hex2Bytes("aaaaabbbbbaaaaabbbbbaaaaabbbbbaa"))
	subTries, err := r.ResolveWithDb(db, 0, rs, [][]byte{common.Hex2Bytes("aaaaaaaaaa")}, []int{40}, false)
	require.NoError(err)

	tr := New(common.Hash{})
	tr.HookSubTries(subTries, [][]byte{nil}) // hook up to the root of the trie
	x, ok := tr.Get(common.Hex2Bytes("aaaaaaaaaaaaaaaaaaaaaaaaaaa"))
	assert.True(ok)
	assert.NotNil(x)
}

func TestResolve2Keep(t *testing.T) {
	t.Skip("weird case of abandoned storage, will handle it later")

	require, assert, db := require.New(t), assert.New(t), ethdb.NewMemDatabase()
	putStorage := func(k string, v string) {
		err := db.Put(dbutils.CurrentStateBucket, common.Hex2Bytes(k), common.Hex2Bytes(v))
		require.NoError(err)
	}
	putStorage("aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", "")
	putStorage("aaaaaccccccccccccccccccccccccccc", "")

	r := NewResolver(0)
	rs := NewResolveSet(0)
	rs.AddKey(common.Hex2Bytes("aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"))
	subTries, err := r.ResolveWithDb(db, 0, rs, [][]byte{common.Hex2Bytes("aaaaaaaaaa")}, []int{40}, false)
	require.NoError(err)

	tr := New(common.Hash{})
	tr.HookSubTries(subTries, [][]byte{nil}) // hook up to the root of the trie
	x, ok := tr.Get(common.Hex2Bytes("aaaaaaaaaaaaaaaaaaaaaaaaaaa"))
	assert.True(ok)
	assert.NotNil(x)
}

func TestResolve3Keep(t *testing.T) {
	t.Skip("weird case of abandoned storage, will handle it later")

	require, assert, db := require.New(t), assert.New(t), ethdb.NewMemDatabase()
	putStorage := func(k string, v string) {
		err := db.Put(dbutils.CurrentStateBucket, common.Hex2Bytes(k), common.Hex2Bytes(v))
		require.NoError(err)
	}
	putStorage("aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", "")
	putStorage("aaaaabbbbbbbbbbbbbbbbbbbbbbbbbbb", "")
	putStorage("aaaaaccccccccccccccccccccccccccc", "")

	r := NewResolver(0)
	rs := NewResolveSet(0)
	rs.AddKey(common.Hex2Bytes("aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"))
	subTries, err := r.ResolveWithDb(db, 0, rs, [][]byte{common.Hex2Bytes("aaaaaaaaaa")}, []int{40}, false)
	require.NoError(err)

	tr := New(common.Hash{})
	tr.HookSubTries(subTries, [][]byte{nil}) // hook up to the root of the trie
	x, ok := tr.Get(common.Hex2Bytes("aaaaaaaaaaaaaaaaaaaaaaaaaaa"))
	assert.True(ok)
	assert.NotNil(x)
}

func TestTrieResolver(t *testing.T) {
	t.Skip("weird case of abandoned storage, will handle it later")

	require, _, db := require.New(t), assert.New(t), ethdb.NewMemDatabase()
	putStorage := func(k string, v string) {
		err := db.Put(dbutils.CurrentStateBucket, common.Hex2Bytes(k), common.Hex2Bytes(v))
		require.NoError(err)
	}
	putStorage("aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", "")
	putStorage("aaaaaccccccccccccccccccccccccccc", "")
	putStorage("baaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", "")
	putStorage("bbaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", "")
	putStorage("bbaaaccccccccccccccccccccccccccc", "")
	putStorage("bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb", "")
	putStorage("bccccccccccccccccccccccccccccccc", "")

	resolver := NewResolver(0)
	rs := NewResolveSet(0)
	rs.AddKey(common.Hex2Bytes("aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"))
	rs.AddKey(common.Hex2Bytes("bbaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"))
	rs.AddKey(common.Hex2Bytes("bbbaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"))
	_, err := resolver.ResolveWithDb(db, 0, rs,
		[][]byte{common.Hex2Bytes("aaaaa"), common.Hex2Bytes("bb")}, []int{40, 8}, false)
	require.NoError(err, "resolve error")
}

func TestTwoStorageItems(t *testing.T) {
	t.Skip("weird case of abandoned storage, will handle it later")

	require, assert, db := require.New(t), assert.New(t), ethdb.NewMemDatabase()

	key1 := common.Hex2Bytes("d7b6990105719101dabeb77144f2a3385c8033acd3af97e9423a695e81ad1eb5f5")
	key2 := common.Hex2Bytes("df6966c971051c3d54ec59162606531493a51404a002842f56009d7e5cf4a8c7f5")
	val1 := common.Hex2Bytes("02")
	val2 := common.Hex2Bytes("03")

	require.NoError(db.Put(dbutils.CurrentStateBucket, key1, val1))
	require.NoError(db.Put(dbutils.CurrentStateBucket, key2, val2))
	leaf1 := shortNode{Key: keybytesToHex(key1[1:]), Val: valueNode(val1)}
	leaf2 := shortNode{Key: keybytesToHex(key2[1:]), Val: valueNode(val2)}
	var branch fullNode
	branch.Children[0x7] = &leaf1
	branch.Children[0xf] = &leaf2
	root := shortNode{Key: []byte{0xd}, Val: &branch}

	hasher := newHasher(false)
	defer returnHasherToPool(hasher)
	rootRlp, err := hasher.hashChildren(&root, 0)
	require.NoError(err, "failed ot hash children")

	// Resolve the root node

	rootHash := common.HexToHash("85737b049107f866fedbd6d787077fc2c245f4748e28896a3e8ee82c377ecdcf")
	assert.Equal(rootHash, crypto.Keccak256Hash(rootRlp))

	resolver := NewResolver(0)
	rs := NewResolveSet(0)
	subTries, err1 := resolver.ResolveWithDb(db, 0, rs, [][]byte{nil}, []int{0}, false)
	require.NoError(err1, "resolve error")

	assert.Equal(rootHash.String(), subTries.Hashes[0].String())
	tr := New(common.Hash{})
	err = tr.HookSubTries(subTries, [][]byte{nil}) // hook up to the root of the trie
	assert.NoError(err)

	// Resolve the branch node

	//branchRlp, err := hasher.hashChildren(&branch, 0)
	//if err != nil {
	//	t.Errorf("failed ot hash children: %v", err)
	//}

	resolver2 := NewResolver(0)
	rs2 := NewResolveSet(0)
	rs2.AddHex([]byte{0xd})
	subTries, err = resolver2.ResolveWithDb(db, 0, rs2, [][]byte{[]byte{0xd0}}, []int{4}, false)
	require.NoError(err, "resolve error")

	err = tr.HookSubTries(subTries, [][]byte{[]byte{0xd}}) // hook up to the prefix 0xd
	assert.NoError(err)

	x, ok := tr.Get(key1)
	assert.True(ok)
	assert.NotNil(x)
}

func TestTwoAccounts(t *testing.T) {
	require, assert, db := require.New(t), assert.New(t), ethdb.NewMemDatabase()
	key1 := common.Hex2Bytes("03601462093b5945d1676df093446790fd31b20e7b12a2e8e5e09d068109616b")
	acc := accounts.NewAccount()
	acc.Initialised = true
	acc.Balance.SetInt64(10000000000)
	acc.CodeHash.SetBytes(common.Hex2Bytes("c5d2460186f7233c927e7db2dcc703c0e500b653ca82273b7bfad8045d85a470"))
	err := writeAccount(db, common.BytesToHash(key1), acc)
	require.NoError(err)

	key2 := common.Hex2Bytes("0fbc62ba90dec43ec1d6016f9dd39dc324e967f2a3459a78281d1f4b2ba962a6")
	acc2 := accounts.NewAccount()
	acc2.Initialised = true
	acc2.Balance.SetInt64(100)
	acc2.CodeHash.SetBytes(common.Hex2Bytes("4f1593970e8f030c0a2c39758181a447774eae7c65653c4e6440e8c18dad69bc"))
	err = writeAccount(db, common.BytesToHash(key2), acc2)
	require.NoError(err)

	expect := common.HexToHash("925002c3260b44e44c3edebad1cc442142b03020209df1ab8bb86752edbd2cd7")

	resolver := NewResolver(0)
	rs := NewResolveSet(0)
	rs.AddKey(key1)
	subTries, err1 := resolver.ResolveWithDb(db, 0, rs, [][]byte{nil}, []int{0}, false)
	require.NoError(err1, "resolve error")
	assert.Equal(expect.String(), subTries.Hashes[0].String())

	tr := New(common.Hash{})
	err = tr.HookSubTries(subTries, [][]byte{nil}) // hook up to the root
	assert.NoError(err)

	x, ok := tr.GetAccount(key1)
	assert.True(ok)
	assert.NotNil(x)
}

func TestReturnErrOnWrongRootHash(t *testing.T) {
	require, db := require.New(t), ethdb.NewMemDatabase()
	putAccount := func(k string) {
		a := accounts.Account{}
		err := writeAccount(db, common.BytesToHash(common.Hex2Bytes(k)), a)
		require.NoError(err)
	}

	putAccount("0000000000000000000000000000000000000000000000000000000000000000")

	rs := NewResolveSet(0)
	resolver := NewResolver(0)
	_, err := resolver.ResolveWithDb(db, 0, rs, [][]byte{nil}, []int{0}, false)
	require.NotNil(t, err)
}

func TestApiDetails(t *testing.T) {
	require, assert, db := require.New(t), assert.New(t), ethdb.NewMemDatabase()

	storageKey := func(incarnation uint64, k string) []byte {
		return dbutils.GenerateCompositeStorageKey(common.HexToHash(k), incarnation, common.HexToHash("0000000000000000000000000000000000000000000000000000000000000001"))
	}
	putIH := func(k string, v string) {
		require.NoError(db.Put(dbutils.IntermediateTrieHashBucket, common.Hex2Bytes(k), common.Hex2Bytes(v)))
	}

	// Test attempt handle cases when: Trie root hash is same for Cached and non-Cached Resolvers
	// Test works with keys like: {base}{i}{j}{zeroes}
	// base = 0 or f - it covers edge cases - first/last subtrees
	//
	// i=0 - has data, has IntermediateHash, no resolve. Tree must have Hash.
	// i=1 - has values with incarnation=1. Tree must have Nil.
	// i=2 - has accounts and storage, no IntermediateHash. Tree must have Account nodes.
	// i>2 - no data, no IntermediateHash, no resolve.
	// i=f - has data, has IntermediateHash, no resolve. Edge case - last subtree.
	for _, base := range []string{"0", "f"} {
		for _, i := range []int{0, 1, 2, 15} {
			for _, j := range []int{0, 1, 2, 15} {
				k := fmt.Sprintf(base+"%x%x%061x", i, j, 0)
				//storageV := common.Hex2Bytes(fmt.Sprintf("%x%x", i, j))
				storageV := []byte{2}
				incarnation := uint64(2)
				root := common.HexToHash("3ed409af5dae7a8b4670f1df4d90d675b115757858b3ea1ec83a323b5c2e060c")
				if i == 1 {
					storageV = []byte{1}
					incarnation = 1
					root = common.HexToHash("2abc098983e097b6de2b7fd9c5c322e37d12025bc3e0cfdaad4af26a3b51e208")
				}

				a := accounts.Account{
					// Using Nonce field as an ID of account.
					// Will check later if value which we .Get() from Trie has expected ID.
					Nonce:          uint64(i*10 + j),
					Initialised:    true,
					Root:           root,
					CodeHash:       EmptyCodeHash,
					Balance:        *big.NewInt(0),
					Incarnation:    incarnation,
				}
				require.NoError(writeAccount(db, common.BytesToHash(common.Hex2Bytes(k)), a))
				require.NoError(db.Put(dbutils.CurrentStateBucket, storageKey(incarnation, k), storageV))
			}
		}
	}

	/*
		Next IH's calculated by next logic:
		var root common.Hash
		_, err = tr.getHasher().hash(tr.root.(*fullNode).Children[0].(*fullNode).Children[0], true, root[:])
		require.NoError(err)
		fmt.Printf("%x\n", root)
	*/
	putIH("00", "83ffec17e97f4d5240cf371a23580225dc086c53ff1b14aad92ededd155a047a")
	putIH("ff", "36548655db5662652d6045bd5474d0d1453122dbc66c55493ed99bcdf9743e7d")

	// this IntermediateHash key must not be used, because such key is in ResolveRequest
	putIH("01", "0000000000000000000000000000000000000000000000000000000000000000")

	{
		resolver := NewResolver(0)
		expectRootHash := common.HexToHash("9a87eff1bc257a70e62dd81ed4b5d210beb305ecdeec9ff497c717ea1c3794d4")

		rs := NewResolveSet(0)
		rs.AddHex(common.Hex2Bytes(fmt.Sprintf("000101%0122x", 0)))
		rs.AddHex(common.Hex2Bytes("000202"))
		rs.AddHex(common.Hex2Bytes("0f"))
		subTries, err := resolver.ResolveWithDb(db, 0, rs, [][]byte{nil}, []int{0}, true)

		//fmt.Printf("%x\n", tr.root.(*fullNode).Children[0].(*fullNode).Children[0].reference())
		//fmt.Printf("%x\n", tr.root.(*fullNode).Children[15].(*fullNode).Children[15].reference())
		assert.NoError(err)

		assert.Equal(expectRootHash.String(), subTries.Hashes[0].String())

		tr := New(common.Hash{})
		err = tr.HookSubTries(subTries, [][]byte{nil}) // hook up to the root
		assert.NoError(err)
		_, found := tr.GetAccount(common.Hex2Bytes(fmt.Sprintf("000%061x", 0)))
		assert.False(found) // exists in DB but resolved, there is hashNode

		acc, found := tr.GetAccount(common.Hex2Bytes(fmt.Sprintf("011%061x", 0)))
		assert.True(found)
		require.NotNil(acc)              // cache bucket has empty value, but self-destructed Account still available
		assert.Equal(int(acc.Nonce), 11) // i * 10 + j

		acc, found = tr.GetAccount(common.Hex2Bytes(fmt.Sprintf("021%061x", 0)))
		assert.True(found)
		require.NotNil(acc)              // exists in db and resolved
		assert.Equal(int(acc.Nonce), 21) // i * 10 + j

		acc, found = tr.GetAccount(common.Hex2Bytes(fmt.Sprintf("051%061x", 0)))
		assert.True(found)
		assert.Nil(acc) // not exists in DB

		assert.Panics(func() {
			tr.UpdateAccount(common.Hex2Bytes(fmt.Sprintf("001%061x", 0)), &accounts.Account{})
		})
		assert.NotPanics(func() {
			tr.UpdateAccount(common.Hex2Bytes(fmt.Sprintf("011%061x", 0)), &accounts.Account{})
			tr.UpdateAccount(common.Hex2Bytes(fmt.Sprintf("021%061x", 0)), &accounts.Account{})
			tr.UpdateAccount(common.Hex2Bytes(fmt.Sprintf("051%061x", 0)), &accounts.Account{})
		})
	}

	/*
		{ // storage resolver
			putIH("00", "0aca8baf23c54bda626bc3c3d1590f9cdb9deb8defaef7455f5f0b55b3d1c76e")
			putIH("ff", "71c0df1d41959526a6961cca7e5831982848074c4cc556fbef4f8a1fad6621ca")

			for i, resolverName := range []string{Stateful, StatefulCached} {
				resolver := NewResolver(32, false, 0)
				expectRootHash := common.HexToHash("494e295f60cfde19548157facc0c425d8b254f791a006b74173dc71113f56df0")

				resolver.AddRequest(tries[i].NewResolveRequest(nil, append(common.Hex2Bytes(fmt.Sprintf("000101%0122x", 0)), 16), 0))
				resolver.AddRequest(tries[i].NewResolveRequest(nil, common.Hex2Bytes("00020100"), 0))
				resolver.AddRequest(tries[i].NewResolveRequest(nil, common.Hex2Bytes("0f"), 0))

				if resolverName == Stateful {
					err := resolver.ResolveStateful(db, 0)
					require.NoError(err)
					//fmt.Printf("%x\n", tr.root.(*fullNode).Children[0].(*fullNode).Children[0].reference())
					//fmt.Printf("%x\n", tr.root.(*fullNode).Children[0].(*fullNode).Children[1].reference())
					_, root := tries[i].DeepHash(common.Hex2Bytes(fmt.Sprintf("021%061x", 0)))
					fmt.Printf("Alex: %x\n", root)
					_, root = tries[i].DeepHash(common.Hex2Bytes(fmt.Spritntf("011%061x", 0)))
					fmt.Printf("Alex: %x\n", root)

					//fmt.Printf("%x\n", tr.root.(*fullNode).Children[15].(*fullNode).Children[15].reference())
				} else {
					err := resolver.ResolveStatefulCached(db, 0, true)
					//fmt.Printf("%x\n", tr.root.(*fullNode).Children[0].(*fullNode).Children[1].reference())
					require.NoError(err)
				}
				//assert.Equal(expectRootHash.String(), tr.Hash().String())

				//_, found := tr.Get(storageKey(2, fmt.Sprintf("000%061x", 0)))
				//assert.False(found) // exists in DB but not resolved, there is hashNode

				storage, found := tries[i].Get(storageKey(2, fmt.Sprintf("011%061x", 0)))
				assert.True(found)
				require.Nil(storage) // deleted by empty value in cache bucket

				//storage, found = tr.Get(storageKey(2, fmt.Sprintf("021%061x", 0)))
				//assert.True(found)
				//require.Equal(storage, common.Hex2Bytes("21"))

				//storage, found = tr.Get(storageKey(2, fmt.Sprintf("051%061x", 0)))
				//assert.True(found)
				//assert.Nil(storage) // not exists in DB

				//assert.Panics(func() {
				//	tr.Update(storageKey(2, fmt.Sprintf("001%061x", 0)), nil)
				//})
				assert.NotPanics(func() {
					tries[i].Update(storageKey(2, fmt.Sprintf("011%061x", 0)), nil)
					tries[i].Update(storageKey(2, fmt.Sprintf("021%061x", 0)), nil)
					tries[i].Update(storageKey(2, fmt.Sprintf("051%061x", 0)), nil)
				})
			}
		}
	*/
}

func TestStorageResolver2(t *testing.T) {
	require, assert, db := require.New(t), assert.New(t), ethdb.NewMemDatabase()

	kAcc1 := common.FromHex("0000cf1ce0664746d39af9f6db99dc3370282f1d9d48df7f804b7e6499558c83")
	a1 := accounts.Account{
		Nonce:          uint64(1),
		Initialised:    true,
		CodeHash:       EmptyCodeHash,
		Balance:        *big.NewInt(0),
		Incarnation:    1,
		Root:           EmptyRoot,
	}

	if err := writeAccount(db, common.BytesToHash(kAcc1), a1); err != nil {
		panic(err)
	}

	kAcc2 := common.FromHex("0001cf1ce0664746d39af9f6db99dc3370282f1d9d48df7f804b7e6499558c83")
	k2 := "290decd9548b62a8d60345a988386fc84ba6bc95484008f6362f93160ef3e563"
	ks2 := dbutils.GenerateCompositeStorageKey(common.BytesToHash(kAcc2), 1, common.HexToHash(k2))
	require.NoError(db.Put(dbutils.CurrentStateBucket, ks2, common.FromHex("7a381122bada791a7ab1f6037dac80432753baad")))

	expectedAccStorageRoot := "28d28aa6f1d0179248560a25a1a4ad69be1cdeab9e2b24bc9f9c70608e3a7ec0"
	expectedAccRoot2 := expectedAccStorageRoot
	a2 := accounts.Account{
		Nonce:          uint64(1),
		Initialised:    true,
		CodeHash:       EmptyCodeHash,
		Balance:        *big.NewInt(0),
		Incarnation:    1,
		Root:           common.HexToHash(expectedAccRoot2),
	}

	if err := writeAccount(db, common.BytesToHash(kAcc2), a2); err != nil {
		panic(err)
	}

	kAcc3 := common.FromHex("0002cf1ce0664746d39af9f6db99dc3370282f1d9d48df7f804b7e6499558c83")
	k3 := k2
	ks3 := dbutils.GenerateCompositeStorageKey(common.BytesToHash(kAcc3), 2, common.HexToHash(k3))
	ks3OldIncarnation := dbutils.GenerateCompositeStorageKey(common.BytesToHash(kAcc3), 1, common.HexToHash(k3))
	require.NoError(db.Put(dbutils.CurrentStateBucket, ks3, common.FromHex("7a381122bada791a7ab1f6037dac80432753baad")))
	require.NoError(db.Put(dbutils.CurrentStateBucket, ks3OldIncarnation, common.FromHex("9999999999999999")))

	expectedAccRoot3 := expectedAccStorageRoot
	a3 := accounts.Account{
		Nonce:          uint64(1),
		Initialised:    true,
		CodeHash:       EmptyCodeHash,
		Balance:        *big.NewInt(0),
		Incarnation:    2,
		Root:           common.HexToHash(expectedAccRoot3),
	}

	if err := writeAccount(db, common.BytesToHash(kAcc3), a3); err != nil {
		panic(err)
	}

	//expectedRoot := "3a9dc9c90290be8d88abea1c01d408e2a4173b4e295863942f0980e49bfbf375"

	// abandoned storage - account was deleted, but storage still exists
	kAcc4 := common.FromHex("0004cf1ce0664746d39af9f6db99dc3370282f1d9d48df7f804b7e6499558c83") // don't write it to db
	ks4 := dbutils.GenerateCompositeStorageKey(common.BytesToHash(kAcc4), 1, common.HexToHash(k2))
	require.NoError(db.Put(dbutils.CurrentStateBucket, ks4, common.FromHex("7a381122bada791a7ab1f6037dac80432753baad")))

	{
		resolver := NewResolver(0)
		rs := NewResolveSet(0)
		rs.AddHex(common.FromHex("00000001"))
		_, err := resolver.ResolveWithDb(db, 0, rs, [][]byte{nil}, []int{0}, false)
		assert.NoError(err)
	}
	{
		resolver := NewResolver(0)
		rs := NewResolveSet(0)
		rs.AddKey(dbutils.GenerateStoragePrefix(kAcc2, a2.Incarnation))
		_, err := resolver.ResolveWithDb(db, 0, rs, [][]byte{nil}, []int{0}, false)
		assert.NoError(err)
	}
}

func TestCreateLoadingPrefixes(t *testing.T) {
	assert := assert.New(t)

	tr := New(common.Hash{})
	kAcc1 := common.FromHex("0001cf1ce0664746d39af9f6db99dc3370282f1d9d48df7f804b7e6499558c83")
	ks1 := common.FromHex("0000000000000000000000000000000000000000000000000000000000000001")
	acc1 := accounts.NewAccount()
	acc1.Balance.SetInt64(12345)
	acc1.Incarnation = 1
	acc1.Initialised = true
	tr.UpdateAccount(kAcc1, &acc1)
	tr.Update(concat(kAcc1, ks1...), []byte{1,2,3})

	kAcc2 := common.FromHex("0002cf1ce0664746d39af9f6db99dc3370282f1d9d48df7f804b7e6499558c83")
	ks2 := common.FromHex("0000000000000000000000000000000000000000000000000000000000000001")
	ks22 := common.FromHex("0000000000000000000000000000000000000000000000000000000000000002")
	acc2 := accounts.NewAccount()
	acc2.Balance.SetInt64(6789)
	acc2.Incarnation = 1
	acc2.Initialised = true
	tr.UpdateAccount(kAcc2, &acc2)
	tr.Update(concat(kAcc2, ks2...), []byte{4,5,6})
	tr.Update(concat(kAcc2, ks22...), []byte{7,8,9})
	tr.Hash()

	// Evict accounts only
	tr.EvictNode(keybytesToHex(kAcc1))
	tr.EvictNode(keybytesToHex(kAcc2))
	rs := NewResolveSet(0)
	rs.AddKey(concat(kAcc1, ks1...))
	rs.AddKey(concat(kAcc2, ks2...))
	rs.AddKey(concat(kAcc2, ks22...))
	dbPrefixes, fixedbits, hooks := tr.CreateLoadingPrefixes(rs)
	assert.Equal("[]", fmt.Sprintf("%x", dbPrefixes))
	assert.Equal("[0]", fmt.Sprintf("%d", fixedbits))
	assert.Equal("[]", fmt.Sprintf("%x", hooks))

	// Evict everytning
	tr.EvictNode([]byte{})
	// if resolve only accounts
	rs = NewResolveSet(0)
	rs.AddKey(kAcc1)
	rs.AddKey(kAcc2)
	dbPrefixes, fixedbits, hooks = tr.CreateLoadingPrefixes(rs)
	assert.Equal("[]", fmt.Sprintf("%x", dbPrefixes))
	assert.Equal("[0]", fmt.Sprintf("%d", fixedbits))
	assert.Equal("[]", fmt.Sprintf("%x", hooks))
}

func TestIsBefore(t *testing.T) {
	assert := assert.New(t)

	is, minKey := keyIsBefore([]byte("a"), []byte("b"))
	assert.Equal(true, is)
	assert.Equal("a", fmt.Sprintf("%s", minKey))

	is, minKey = keyIsBefore([]byte("b"), []byte("a"))
	assert.Equal(false, is)
	assert.Equal("a", fmt.Sprintf("%s", minKey))

	is, minKey = keyIsBefore([]byte("b"), []byte(""))
	assert.Equal(false, is)
	assert.Equal("", fmt.Sprintf("%s", minKey))

	is, minKey = keyIsBefore(nil, []byte("b"))
	assert.Equal(false, is)
	assert.Equal("b", fmt.Sprintf("%s", minKey))

	is, minKey = keyIsBefore([]byte("b"), nil)
	assert.Equal(true, is)
	assert.Equal("b", fmt.Sprintf("%s", minKey))

	contract := fmt.Sprintf("2%063x", 0)
	storageKey := common.Hex2Bytes(contract + "ffffffff" + fmt.Sprintf("10%062x", 0))
	cacheKey := common.Hex2Bytes(contract + "ffffffff" + "20")
	is, minKey = keyIsBefore(cacheKey, storageKey)
	assert.False(is)
	assert.Equal(fmt.Sprintf("%x", storageKey), fmt.Sprintf("%x", minKey))

	storageKey = common.Hex2Bytes(contract + "ffffffffffffffff" + fmt.Sprintf("20%062x", 0))
	cacheKey = common.Hex2Bytes(contract + "ffffffffffffffff" + "10")
	is, minKey = keyIsBefore(cacheKey, storageKey)
	assert.True(is)
	assert.Equal(fmt.Sprintf("%x", cacheKey), fmt.Sprintf("%x", minKey))
}

func writeAccount(db ethdb.Putter, addrHash common.Hash, acc accounts.Account) error {
	value := make([]byte, acc.EncodingLengthForStorage())
	acc.EncodeForStorage(value)
	if err := db.Put(dbutils.CurrentStateBucket, addrHash[:], value); err != nil {
		return err
	}
	if err := db.Put(dbutils.IntermediateTrieHashBucket, dbutils.GenerateStoragePrefix(addrHash[:], acc.Incarnation), acc.Root.Bytes()); err != nil {
		return err
	}
	return nil
}
