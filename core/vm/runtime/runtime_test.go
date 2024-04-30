// Copyright 2015 The go-ethereum Authors
// This file is part of the go-ethereum library.
//
// The go-ethereum library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-ethereum library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-ethereum library. If not, see <http://www.gnu.org/licenses/>.

package runtime

import (
	"context"
	"fmt"
	"math/big"
	"os"
	"strings"
	"testing"

	"github.com/ledgerwatch/erigon-lib/chain"
	libcommon "github.com/ledgerwatch/erigon-lib/common"
	"github.com/ledgerwatch/erigon-lib/kv/memdb"
	"github.com/ledgerwatch/erigon/accounts/abi"
	"github.com/ledgerwatch/erigon/common"
	"github.com/ledgerwatch/erigon/consensus"
	"github.com/ledgerwatch/erigon/consensus/misc"
	"github.com/ledgerwatch/erigon/core"
	"github.com/ledgerwatch/erigon/core/asm"
	"github.com/ledgerwatch/erigon/core/state"
	"github.com/ledgerwatch/erigon/core/types"
	"github.com/ledgerwatch/erigon/core/vm"
	"github.com/ledgerwatch/erigon/eth/tracers/logger"
	"github.com/ledgerwatch/erigon/params"
	"github.com/ledgerwatch/erigon/rlp"
)

func TestDefaults(t *testing.T) {
	t.Parallel()
	cfg := new(Config)
	setDefaults(cfg)

	if cfg.Difficulty == nil {
		t.Error("expected difficulty to be non nil")
	}

	if cfg.Time == nil {
		t.Error("expected time to be non nil")
	}
	if cfg.GasLimit == 0 {
		t.Error("didn't expect gaslimit to be zero")
	}
	if cfg.GasPrice == nil {
		t.Error("expected time to be non nil")
	}
	if cfg.Value == nil {
		t.Error("expected time to be non nil")
	}
	if cfg.GetHashFn == nil {
		t.Error("expected time to be non nil")
	}
	if cfg.BlockNumber == nil {
		t.Error("expected block number to be non nil")
	}
}

func TestEVM(t *testing.T) {
	t.Parallel()
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("crashed with: %v", r)
		}
	}()

	if _, _, err := Execute([]byte{
		byte(vm.DIFFICULTY),
		byte(vm.TIMESTAMP),
		byte(vm.GASLIMIT),
		byte(vm.PUSH1),
		byte(vm.ORIGIN),
		byte(vm.BLOCKHASH),
		byte(vm.COINBASE),
	}, nil, nil, 0); err != nil {
		t.Fatal("didn't expect error", err)
	}
}

func TestExecute(t *testing.T) {
	t.Parallel()
	ret, _, err := Execute([]byte{
		byte(vm.PUSH1), 10,
		byte(vm.PUSH1), 0,
		byte(vm.MSTORE),
		byte(vm.PUSH1), 32,
		byte(vm.PUSH1), 0,
		byte(vm.RETURN),
	}, nil, nil, 0)
	if err != nil {
		t.Fatal("didn't expect error", err)
	}

	num := new(big.Int).SetBytes(ret)
	if num.Cmp(big.NewInt(10)) != 0 {
		t.Error("Expected 10, got", num)
	}
}

func TestCall(t *testing.T) {
	t.Parallel()
	_, tx := memdb.NewTestTx(t)
	state := state.New(state.NewDbStateReader(tx))
	address := libcommon.HexToAddress("0xaa")
	state.SetCode(address, []byte{
		byte(vm.PUSH1), 10,
		byte(vm.PUSH1), 0,
		byte(vm.MSTORE),
		byte(vm.PUSH1), 32,
		byte(vm.PUSH1), 0,
		byte(vm.RETURN),
	})

	ret, _, err := Call(address, nil, &Config{State: state})
	if err != nil {
		t.Fatal("didn't expect error", err)
	}

	num := new(big.Int).SetBytes(ret)
	if num.Cmp(big.NewInt(10)) != 0 {
		t.Error("Expected 10, got", num)
	}
}

func BenchmarkCall(b *testing.B) {
	var definition = `[{"constant":true,"inputs":[],"name":"seller","outputs":[{"name":"","type":"address"}],"type":"function"},{"constant":false,"inputs":[],"name":"abort","outputs":[],"type":"function"},{"constant":true,"inputs":[],"name":"value","outputs":[{"name":"","type":"uint256"}],"type":"function"},{"constant":false,"inputs":[],"name":"refund","outputs":[],"type":"function"},{"constant":true,"inputs":[],"name":"buyer","outputs":[{"name":"","type":"address"}],"type":"function"},{"constant":false,"inputs":[],"name":"confirmReceived","outputs":[],"type":"function"},{"constant":true,"inputs":[],"name":"state","outputs":[{"name":"","type":"uint8"}],"type":"function"},{"constant":false,"inputs":[],"name":"confirmPurchase","outputs":[],"type":"function"},{"inputs":[],"type":"constructor"},{"anonymous":false,"inputs":[],"name":"Aborted","type":"event"},{"anonymous":false,"inputs":[],"name":"PurchaseConfirmed","type":"event"},{"anonymous":false,"inputs":[],"name":"ItemReceived","type":"event"},{"anonymous":false,"inputs":[],"name":"Refunded","type":"event"}]`

	var code = libcommon.Hex2Bytes("6060604052361561006c5760e060020a600035046308551a53811461007457806335a063b4146100865780633fa4f245146100a6578063590e1ae3146100af5780637150d8ae146100cf57806373fac6f0146100e1578063c19d93fb146100fe578063d696069714610112575b610131610002565b610133600154600160a060020a031681565b610131600154600160a060020a0390811633919091161461015057610002565b61014660005481565b610131600154600160a060020a039081163391909116146102d557610002565b610133600254600160a060020a031681565b610131600254600160a060020a0333811691161461023757610002565b61014660025460ff60a060020a9091041681565b61013160025460009060ff60a060020a9091041681146101cc57610002565b005b600160a060020a03166060908152602090f35b6060908152602090f35b60025460009060a060020a900460ff16811461016b57610002565b600154600160a060020a03908116908290301631606082818181858883f150506002805460a060020a60ff02191660a160020a179055506040517f72c874aeff0b183a56e2b79c71b46e1aed4dee5e09862134b8821ba2fddbf8bf9250a150565b80546002023414806101dd57610002565b6002805460a060020a60ff021973ffffffffffffffffffffffffffffffffffffffff1990911633171660a060020a1790557fd5d55c8a68912e9a110618df8d5e2e83b8d83211c57a8ddd1203df92885dc881826060a15050565b60025460019060a060020a900460ff16811461025257610002565b60025460008054600160a060020a0390921691606082818181858883f150508354604051600160a060020a0391821694503090911631915082818181858883f150506002805460a060020a60ff02191660a160020a179055506040517fe89152acd703c9d8c7d28829d443260b411454d45394e7995815140c8cbcbcf79250a150565b60025460019060a060020a900460ff1681146102f057610002565b6002805460008054600160a060020a0390921692909102606082818181858883f150508354604051600160a060020a0391821694503090911631915082818181858883f150506002805460a060020a60ff02191660a160020a179055506040517f8616bbbbad963e4e65b1366f1d75dfb63f9e9704bbbf91fb01bec70849906cf79250a15056")

	abi, err := abi.JSON(strings.NewReader(definition))
	if err != nil {
		b.Fatal(err)
	}

	cpurchase, err := abi.Pack("confirmPurchase")
	if err != nil {
		b.Fatal(err)
	}
	creceived, err := abi.Pack("confirmReceived")
	if err != nil {
		b.Fatal(err)
	}
	refund, err := abi.Pack("refund")
	if err != nil {
		b.Fatal(err)
	}
	cfg := &Config{}
	db := memdb.New("")
	defer db.Close()
	tx, err := db.BeginRw(context.Background())
	if err != nil {
		panic(err)
	}
	defer tx.Rollback()
	cfg.r = state.NewPlainStateReader(tx)
	cfg.w = state.NewPlainStateWriter(tx, tx, 0)
	cfg.State = state.New(cfg.r)

	cfg.Debug = true
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for j := 0; j < 400; j++ {
			_, _, _ = Execute(code, cpurchase, cfg, 0)
			_, _, _ = Execute(code, creceived, cfg, 0)
			_, _, _ = Execute(code, refund, cfg, 0)
		}
	}
}
func benchmarkEVM_Create(b *testing.B, code string) {
	_, tx := memdb.NewTestTx(b)
	var (
		statedb  = state.New(state.NewPlainState(tx, 1, nil))
		sender   = libcommon.BytesToAddress([]byte("sender"))
		receiver = libcommon.BytesToAddress([]byte("receiver"))
	)

	statedb.CreateAccount(sender, true)
	statedb.SetCode(receiver, common.FromHex(code))
	runtimeConfig := Config{
		Origin:      sender,
		State:       statedb,
		GasLimit:    10000000,
		Difficulty:  big.NewInt(0x200000),
		Time:        new(big.Int).SetUint64(0),
		Coinbase:    libcommon.Address{},
		BlockNumber: new(big.Int).SetUint64(1),
		ChainConfig: &chain.Config{
			ChainID:               big.NewInt(1),
			HomesteadBlock:        new(big.Int),
			ByzantiumBlock:        new(big.Int),
			ConstantinopleBlock:   new(big.Int),
			TangerineWhistleBlock: new(big.Int),
			SpuriousDragonBlock:   new(big.Int),
		},
		EVMConfig: vm.Config{},
	}
	// Warm up the intpools and stuff
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _, _ = Call(receiver, []byte{}, &runtimeConfig)
	}
	b.StopTimer()
}

func BenchmarkEVM_CREATE_500(bench *testing.B) {
	// initcode size 500K, repeatedly calls CREATE and then modifies the mem contents
	benchmarkEVM_Create(bench, "5b6207a120600080f0600152600056")
}
func BenchmarkEVM_CREATE2_500(bench *testing.B) {
	// initcode size 500K, repeatedly calls CREATE2 and then modifies the mem contents
	benchmarkEVM_Create(bench, "5b586207a120600080f5600152600056")
}
func BenchmarkEVM_CREATE_1200(bench *testing.B) {
	// initcode size 1200K, repeatedly calls CREATE and then modifies the mem contents
	benchmarkEVM_Create(bench, "5b62124f80600080f0600152600056")
}
func BenchmarkEVM_CREATE2_1200(bench *testing.B) {
	// initcode size 1200K, repeatedly calls CREATE2 and then modifies the mem contents
	benchmarkEVM_Create(bench, "5b5862124f80600080f5600152600056")
}

func fakeHeader(n uint64, parentHash libcommon.Hash) *types.Header {
	header := types.Header{
		Coinbase:   libcommon.HexToAddress("0x00000000000000000000000000000000deadbeef"),
		Number:     big.NewInt(int64(n)),
		ParentHash: parentHash,
		Time:       n,
		Nonce:      types.BlockNonce{0x1},
		Extra:      []byte{},
		Difficulty: big.NewInt(0),
		GasLimit:   100000,
	}
	return &header
}

type FakeChainHeaderReader struct {
	// Cfg     *chain.Config
	// current *types.Block
}

func (cr *FakeChainHeaderReader) GetHeaderByHash(hash libcommon.Hash) *types.Header {
	num := hash.Big()
	return &types.Header{
		Coinbase:   libcommon.HexToAddress("0x00000000000000000000000000000000deadbeef"),
		Number:     num,
		ParentHash: libcommon.BigToHash(big.NewInt(0).Sub(num, big.NewInt(1))),
		Time:       num.Uint64(),
		Nonce:      types.BlockNonce{0x1},
		Extra:      []byte{},
		Difficulty: big.NewInt(0),
		GasLimit:   100000,
	}
}
func (cr *FakeChainHeaderReader) GetHeaderByNumber(number uint64) *types.Header {
	return cr.GetHeaderByHash(libcommon.BigToHash(big.NewInt(int64(number))))
}
func (cr *FakeChainHeaderReader) Config() *chain.Config        { return nil }
func (cr *FakeChainHeaderReader) CurrentHeader() *types.Header { return nil }
func (cr *FakeChainHeaderReader) GetHeader(hash libcommon.Hash, number uint64) *types.Header {
	return nil
}
func (cr *FakeChainHeaderReader) GetBlock(hash libcommon.Hash, number uint64) *types.Block {
	return nil
}
func (cr *FakeChainHeaderReader) HasBlock(hash libcommon.Hash, number uint64) bool  { return false }
func (cr *FakeChainHeaderReader) GetTd(hash libcommon.Hash, number uint64) *big.Int { return nil }
func (cr *FakeChainHeaderReader) FrozenBlocks() uint64                              { return 0 }
func (cr *FakeChainHeaderReader) BorEventsByBlock(hash libcommon.Hash, number uint64) []rlp.RawValue {
	return nil
}
func (cr *FakeChainHeaderReader) BorStartEventID(hash libcommon.Hash, number uint64) uint64 {
	return 0
}
func (cr *FakeChainHeaderReader) BorSpan(spanId uint64) []byte { return nil }

type dummyChain struct {
	counter int
}

// Engine retrieves the chain's consensus engine.
func (d *dummyChain) Engine() consensus.Engine {
	return nil
}

// GetHeader returns the hash corresponding to their hash.
func (d *dummyChain) GetHeader(h libcommon.Hash, n uint64) *types.Header {
	d.counter++
	parentHash := libcommon.Hash{}
	s := common.LeftPadBytes(big.NewInt(int64(n-1)).Bytes(), 32)
	copy(parentHash[:], s)

	//parentHash := libcommon.Hash{byte(n - 1)}
	//fmt.Printf("GetHeader(%x, %d) => header with parent %x\n", h, n, parentHash)
	return fakeHeader(n, parentHash)
}

// TestBlockhash tests the blockhash operation. It's a bit special, since it internally
// requires access to a chain reader.
func TestBlockhash(t *testing.T) {
	t.Parallel()
	// Current head
	n := uint64(1000)
	parentHash := libcommon.Hash{}
	s := common.LeftPadBytes(big.NewInt(int64(n-1)).Bytes(), 32)
	copy(parentHash[:], s)
	header := fakeHeader(n, parentHash)

	// This is the contract we're using. It requests the blockhash for current num (should be all zeroes),
	// then iteratively fetches all blockhashes back to n-260.
	// It returns
	// 1. the first (should be zero)
	// 2. the second (should be the parent hash)
	// 3. the last non-zero hash
	// By making the chain reader return hashes which correlate to the number, we can
	// verify that it obtained the right hashes where it should

	/*

		pragma solidity ^0.5.3;
		contract Hasher{

			function test() public view returns (bytes32, bytes32, bytes32){
				uint256 x = block.number;
				bytes32 first;
				bytes32 last;
				bytes32 zero;
				zero = blockhash(x); // Should be zeroes
				first = blockhash(x-1);
				for(uint256 i = 2 ; i < 260; i++){
					bytes32 hash = blockhash(x - i);
					if (uint256(hash) != 0){
						last = hash;
					}
				}
				return (zero, first, last);
			}
		}

	*/
	// The contract above
	data := libcommon.Hex2Bytes("6080604052348015600f57600080fd5b50600436106045576000357c010000000000000000000000000000000000000000000000000000000090048063f8a8fd6d14604a575b600080fd5b60506074565b60405180848152602001838152602001828152602001935050505060405180910390f35b600080600080439050600080600083409050600184034092506000600290505b61010481101560c35760008186034090506000816001900414151560b6578093505b5080806001019150506094565b508083839650965096505050505090919256fea165627a7a72305820dfe2458cc086a06042239d07ce304746e65c88f8a42a9b5dfb0ac053c287d8380029")
	// The method call to 'test()'
	input := libcommon.Hex2Bytes("f8a8fd6d")
	chain := &dummyChain{}
	cfg := &Config{
		GetHashFn:   core.GetHashFn(header, chain.GetHeader),
		BlockNumber: new(big.Int).Set(header.Number),
		Time:        new(big.Int),
	}
	setDefaults(cfg)
	ret, _, err := Execute(data, input, cfg, header.Number.Uint64())
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(ret) != 96 {
		t.Fatalf("expected returndata to be 96 bytes, got %d", len(ret))
	}

	zero := new(big.Int).SetBytes(ret[0:32])
	first := new(big.Int).SetBytes(ret[32:64])
	last := new(big.Int).SetBytes(ret[64:96])
	if zero.Sign() != 0 {
		t.Fatalf("expected zeroes, got %x", ret[0:32])
	}
	if first.Uint64() != 999 {
		t.Fatalf("second block should be 999, got %d (%x)", first, ret[32:64])
	}
	if last.Uint64() != 744 {
		t.Fatalf("last block should be 744, got %d (%x)", last, ret[64:96])
	}
	if exp, got := 255, chain.counter; exp != got {
		t.Errorf("suboptimal; too much chain iteration, expected %d, got %d", exp, got)
	}
}

func TestBlockHashEip2935(t *testing.T) {
	t.Parallel()

	// This is the contract we're using. It requests the blockhash for current num (should be all zeroes), We are fetching BlockHash for current block (should be zer0), parent block, last block which is supposed to be there (head - HISTORY_SERVE_WINDOW) and also one block before that (should be zero)

	/*
		pragma solidity ^0.8.25;
		contract BlockHashTestPrague{
			function test() public view returns (bytes32, bytes32, bytes32, bytes32){
				uint256 head = block.number;
				bytes32 zero = blockhash(head);
				bytes32 first = blockhash(head-1);
				bytes32 last = blockhash(head - 8192);
				bytes32 beyond = blockhash(head - 8193);
				return (zero, first, last, beyond);
			}
		}
	*/
	// The contract above
	data := libcommon.Hex2Bytes("608060405234801561000f575f80fd5b5060043610610029575f3560e01c8063f8a8fd6d1461002d575b5f80fd5b61003561004e565b60405161004594939291906100bf565b60405180910390f35b5f805f805f4390505f814090505f6001836100699190610138565b4090505f6120008461007b9190610138565b4090505f6120018561008d9190610138565b409050838383839850985098509850505050505090919293565b5f819050919050565b6100b9816100a7565b82525050565b5f6080820190506100d25f8301876100b0565b6100df60208301866100b0565b6100ec60408301856100b0565b6100f960608301846100b0565b95945050505050565b5f819050919050565b7f4e487b71000000000000000000000000000000000000000000000000000000005f52601160045260245ffd5b5f61014282610102565b915061014d83610102565b92508282039050818111156101655761016461010b565b5b9291505056fea2646970667358221220bac67d00c05154c1dca13fe3c1493172d44692d312cb3fd72a3d7457874d595464736f6c63430008190033")
	// The method call to 'test()'
	input := libcommon.Hex2Bytes("f8a8fd6d")

	// Current head
	n := uint64(10000)
	parentHash := libcommon.Hash{}
	s := common.LeftPadBytes(big.NewInt(int64(n-1)).Bytes(), 32)
	copy(parentHash[:], s)
	fakeHeaderReader := &FakeChainHeaderReader{}
	header := fakeHeaderReader.GetHeaderByNumber(n)

	chain := &dummyChain{}
	cfg := &Config{
		GetHashFn:   core.GetHashFn(header, chain.GetHeader),
		BlockNumber: new(big.Int).Set(header.Number),
		Time:        big.NewInt(10000),
	}
	setDefaults(cfg)
	cfg.ChainConfig.PragueTime = big.NewInt(10000)
	_, tx := memdb.NewTestTx(t)
	cfg.State = state.New(state.NewPlainStateReader(tx))
	cfg.State.CreateAccount(params.HistoryStorageAddress, true)
	misc.StoreBlockHashesEip2935(header, cfg.State, cfg.ChainConfig, &FakeChainHeaderReader{})

	ret, _, err := Execute(data, input, cfg, header.Number.Uint64())
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(ret) != 128 {
		t.Fatalf("expected returndata to be 128 bytes, got %d", len(ret))
	}

	zero := new(big.Int).SetBytes(ret[0:32])
	first := new(big.Int).SetBytes(ret[32:64])
	last := new(big.Int).SetBytes(ret[64:96])
	beyond := new(big.Int).SetBytes(ret[96:128])
	if zero.Sign() != 0 || beyond.Sign() != 0 {
		t.Fatalf("expected zeroes, got %x %x", ret[0:32], ret[96:128])
	}
	if first.Uint64() != 9999 {
		t.Fatalf("second block should be 9999, got %d (%x)", first, ret[32:64])
	}
	if last.Uint64() != 1808 {
		t.Fatalf("last block should be 1808, got %d (%x)", last, ret[64:96])
	}
}

// benchmarkNonModifyingCode benchmarks code, but if the code modifies the
// state, this should not be used, since it does not reset the state between runs.
func benchmarkNonModifyingCode(b *testing.B, gas uint64, code []byte, name string) { //nolint:unparam
	cfg := new(Config)
	setDefaults(cfg)
	_, tx := memdb.NewTestTx(b)
	cfg.State = state.New(state.NewPlainState(tx, 1, nil))
	cfg.GasLimit = gas
	var (
		destination = libcommon.BytesToAddress([]byte("contract"))
		vmenv       = NewEnv(cfg)
		sender      = vm.AccountRef(cfg.Origin)
	)
	cfg.State.CreateAccount(destination, true)
	eoa := libcommon.HexToAddress("E0")
	{
		cfg.State.CreateAccount(eoa, true)
		cfg.State.SetNonce(eoa, 100)
	}
	reverting := libcommon.HexToAddress("EE")
	{
		cfg.State.CreateAccount(reverting, true)
		cfg.State.SetCode(reverting, []byte{
			byte(vm.PUSH1), 0x00,
			byte(vm.PUSH1), 0x00,
			byte(vm.REVERT),
		})
	}

	//cfg.State.CreateAccount(cfg.Origin)
	// set the receiver's (the executing contract) code for execution.
	cfg.State.SetCode(destination, code)
	vmenv.Call(sender, destination, nil, gas, cfg.Value, false /* bailout */) // nolint:errcheck

	b.Run(name, func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			vmenv.Call(sender, destination, nil, gas, cfg.Value, false /* bailout */) // nolint:errcheck
		}
	})
}

// BenchmarkSimpleLoop test a pretty simple loop which loops until OOG
// 55 ms
func BenchmarkSimpleLoop(b *testing.B) {

	staticCallIdentity := []byte{
		byte(vm.JUMPDEST), //  [ count ]
		// push args for the call
		byte(vm.PUSH1), 0, // out size
		byte(vm.DUP1),       // out offset
		byte(vm.DUP1),       // out insize
		byte(vm.DUP1),       // in offset
		byte(vm.PUSH1), 0x4, // address of identity
		byte(vm.GAS), // gas
		byte(vm.STATICCALL),
		byte(vm.POP),      // pop return value
		byte(vm.PUSH1), 0, // jumpdestination
		byte(vm.JUMP),
	}

	callIdentity := []byte{
		byte(vm.JUMPDEST), //  [ count ]
		// push args for the call
		byte(vm.PUSH1), 0, // out size
		byte(vm.DUP1),       // out offset
		byte(vm.DUP1),       // out insize
		byte(vm.DUP1),       // in offset
		byte(vm.DUP1),       // value
		byte(vm.PUSH1), 0x4, // address of identity
		byte(vm.GAS), // gas
		byte(vm.CALL),
		byte(vm.POP),      // pop return value
		byte(vm.PUSH1), 0, // jumpdestination
		byte(vm.JUMP),
	}

	callInexistant := []byte{
		byte(vm.JUMPDEST), //  [ count ]
		// push args for the call
		byte(vm.PUSH1), 0, // out size
		byte(vm.DUP1),        // out offset
		byte(vm.DUP1),        // out insize
		byte(vm.DUP1),        // in offset
		byte(vm.DUP1),        // value
		byte(vm.PUSH1), 0xff, // address of existing contract
		byte(vm.GAS), // gas
		byte(vm.CALL),
		byte(vm.POP),      // pop return value
		byte(vm.PUSH1), 0, // jumpdestination
		byte(vm.JUMP),
	}

	callEOA := []byte{
		byte(vm.JUMPDEST), //  [ count ]
		// push args for the call
		byte(vm.PUSH1), 0, // out size
		byte(vm.DUP1),        // out offset
		byte(vm.DUP1),        // out insize
		byte(vm.DUP1),        // in offset
		byte(vm.DUP1),        // value
		byte(vm.PUSH1), 0xE0, // address of EOA
		byte(vm.GAS), // gas
		byte(vm.CALL),
		byte(vm.POP),      // pop return value
		byte(vm.PUSH1), 0, // jumpdestination
		byte(vm.JUMP),
	}

	loopingCode := []byte{
		byte(vm.JUMPDEST), //  [ count ]
		// push args for the call
		byte(vm.PUSH1), 0, // out size
		byte(vm.DUP1),       // out offset
		byte(vm.DUP1),       // out insize
		byte(vm.DUP1),       // in offset
		byte(vm.PUSH1), 0x4, // address of identity
		byte(vm.GAS), // gas

		byte(vm.POP), byte(vm.POP), byte(vm.POP), byte(vm.POP), byte(vm.POP), byte(vm.POP),
		byte(vm.PUSH1), 0, // jumpdestination
		byte(vm.JUMP),
	}

	calllRevertingContractWithInput := []byte{
		byte(vm.JUMPDEST), //
		// push args for the call
		byte(vm.PUSH1), 0, // out size
		byte(vm.DUP1),        // out offset
		byte(vm.PUSH1), 0x20, // in size
		byte(vm.PUSH1), 0x00, // in offset
		byte(vm.PUSH1), 0x00, // value
		byte(vm.PUSH1), 0xEE, // address of reverting contract
		byte(vm.GAS), // gas
		byte(vm.CALL),
		byte(vm.POP),      // pop return value
		byte(vm.PUSH1), 0, // jumpdestination
		byte(vm.JUMP),
	}

	//tracer := vm.NewJSONLogger(nil, os.Stdout)
	//Execute(loopingCode, nil, &Config{
	//	EVMConfig: vm.Config{
	//		Debug:  true,
	//		Tracer: tracer,
	//	}})
	// 100M gas
	benchmarkNonModifyingCode(b, 100000000, staticCallIdentity, "staticcall-identity-100M")
	benchmarkNonModifyingCode(b, 100000000, callIdentity, "call-identity-100M")
	benchmarkNonModifyingCode(b, 100000000, loopingCode, "loop-100M")
	benchmarkNonModifyingCode(b, 100000000, callInexistant, "call-nonexist-100M")
	benchmarkNonModifyingCode(b, 100000000, callEOA, "call-EOA-100M")
	benchmarkNonModifyingCode(b, 100000000, calllRevertingContractWithInput, "call-reverting-100M")

	//benchmarkNonModifyingCode(10000000, staticCallIdentity, "staticcall-identity-10M", b)
	//benchmarkNonModifyingCode(10000000, loopingCode, "loop-10M", b)
}

// TestEip2929Cases contains various testcases that are used for
// EIP-2929 about gas repricings
func TestEip2929Cases(t *testing.T) {

	id := 1
	prettyPrint := func(comment string, code []byte) {

		instrs := make([]string, 0)
		it := asm.NewInstructionIterator(code)
		for it.Next() {
			if it.Arg() != nil && 0 < len(it.Arg()) {
				instrs = append(instrs, fmt.Sprintf("%v 0x%x", it.Op(), it.Arg()))
			} else {
				instrs = append(instrs, fmt.Sprintf("%v", it.Op()))
			}
		}
		ops := strings.Join(instrs, ", ")
		fmt.Printf("### Case %d\n\n", id)
		id++
		fmt.Printf("%v\n\nBytecode: \n```\n0x%x\n```\nOperations: \n```\n%v\n```\n\n",
			comment,
			code, ops)
		cfg := &Config{
			EVMConfig: vm.Config{
				Debug:     true,
				Tracer:    logger.NewMarkdownLogger(nil, os.Stdout),
				ExtraEips: []int{2929},
			},
		}
		setDefaults(cfg)
		//nolint:errcheck
		Execute(code, nil, cfg, 0)
	}

	{ // First eip testcase
		code := []byte{
			// Three checks against a precompile
			byte(vm.PUSH1), 1, byte(vm.EXTCODEHASH), byte(vm.POP),
			byte(vm.PUSH1), 2, byte(vm.EXTCODESIZE), byte(vm.POP),
			byte(vm.PUSH1), 3, byte(vm.BALANCE), byte(vm.POP),
			// Three checks against a non-precompile
			byte(vm.PUSH1), 0xf1, byte(vm.EXTCODEHASH), byte(vm.POP),
			byte(vm.PUSH1), 0xf2, byte(vm.EXTCODESIZE), byte(vm.POP),
			byte(vm.PUSH1), 0xf3, byte(vm.BALANCE), byte(vm.POP),
			// Same three checks (should be cheaper)
			byte(vm.PUSH1), 0xf2, byte(vm.EXTCODEHASH), byte(vm.POP),
			byte(vm.PUSH1), 0xf3, byte(vm.EXTCODESIZE), byte(vm.POP),
			byte(vm.PUSH1), 0xf1, byte(vm.BALANCE), byte(vm.POP),
			// Check the origin, and the 'this'
			byte(vm.ORIGIN), byte(vm.BALANCE), byte(vm.POP),
			byte(vm.ADDRESS), byte(vm.BALANCE), byte(vm.POP),

			byte(vm.STOP),
		}
		prettyPrint("This checks `EXT`(codehash,codesize,balance) of precompiles, which should be `100`, "+
			"and later checks the same operations twice against some non-precompiles. "+
			"Those are cheaper second time they are accessed. Lastly, it checks the `BALANCE` of `origin` and `this`.", code)
	}

	{ // EXTCODECOPY
		code := []byte{
			// extcodecopy( 0xff,0,0,0,0)
			byte(vm.PUSH1), 0x00, byte(vm.PUSH1), 0x00, byte(vm.PUSH1), 0x00, //length, codeoffset, memoffset
			byte(vm.PUSH1), 0xff, byte(vm.EXTCODECOPY),
			// extcodecopy( 0xff,0,0,0,0)
			byte(vm.PUSH1), 0x00, byte(vm.PUSH1), 0x00, byte(vm.PUSH1), 0x00, //length, codeoffset, memoffset
			byte(vm.PUSH1), 0xff, byte(vm.EXTCODECOPY),
			// extcodecopy( this,0,0,0,0)
			byte(vm.PUSH1), 0x00, byte(vm.PUSH1), 0x00, byte(vm.PUSH1), 0x00, //length, codeoffset, memoffset
			byte(vm.ADDRESS), byte(vm.EXTCODECOPY),

			byte(vm.STOP),
		}
		prettyPrint("This checks `extcodecopy( 0xff,0,0,0,0)` twice, (should be expensive first time), "+
			"and then does `extcodecopy( this,0,0,0,0)`.", code)
	}

	{ // SLOAD + SSTORE
		code := []byte{

			// Add slot `0x1` to access list
			byte(vm.PUSH1), 0x01, byte(vm.SLOAD), byte(vm.POP), // SLOAD( 0x1) (add to access list)
			// Write to `0x1` which is already in access list
			byte(vm.PUSH1), 0x11, byte(vm.PUSH1), 0x01, byte(vm.SSTORE), // SSTORE( loc: 0x01, val: 0x11)
			// Write to `0x2` which is not in access list
			byte(vm.PUSH1), 0x11, byte(vm.PUSH1), 0x02, byte(vm.SSTORE), // SSTORE( loc: 0x02, val: 0x11)
			// Write again to `0x2`
			byte(vm.PUSH1), 0x11, byte(vm.PUSH1), 0x02, byte(vm.SSTORE), // SSTORE( loc: 0x02, val: 0x11)
			// Read slot in access list (0x2)
			byte(vm.PUSH1), 0x02, byte(vm.SLOAD), // SLOAD( 0x2)
			// Read slot in access list (0x1)
			byte(vm.PUSH1), 0x01, byte(vm.SLOAD), // SLOAD( 0x1)
		}
		prettyPrint("This checks `sload( 0x1)` followed by `sstore(loc: 0x01, val:0x11)`, then 'naked' sstore:"+
			"`sstore(loc: 0x02, val:0x11)` twice, and `sload(0x2)`, `sload(0x1)`. ", code)
	}
	{ // Call variants
		code := []byte{
			// identity precompile
			byte(vm.PUSH1), 0x0, byte(vm.DUP1), byte(vm.DUP1), byte(vm.DUP1), byte(vm.DUP1),
			byte(vm.PUSH1), 0x04, byte(vm.PUSH1), 0x0, byte(vm.CALL), byte(vm.POP),

			// random account - call 1
			byte(vm.PUSH1), 0x0, byte(vm.DUP1), byte(vm.DUP1), byte(vm.DUP1), byte(vm.DUP1),
			byte(vm.PUSH1), 0xff, byte(vm.PUSH1), 0x0, byte(vm.CALL), byte(vm.POP),

			// random account - call 2
			byte(vm.PUSH1), 0x0, byte(vm.DUP1), byte(vm.DUP1), byte(vm.DUP1), byte(vm.DUP1),
			byte(vm.PUSH1), 0xff, byte(vm.PUSH1), 0x0, byte(vm.STATICCALL), byte(vm.POP),
		}
		prettyPrint("This calls the `identity`-precompile (cheap), then calls an account (expensive) and `staticcall`s the same"+
			"account (cheap)", code)
	}
}
