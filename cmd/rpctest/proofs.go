package main

import (
	"fmt"
	"net/http"
	"time"

	"github.com/ledgerwatch/turbo-geth/common"
	"github.com/ledgerwatch/turbo-geth/common/hexutil"
	"github.com/ledgerwatch/turbo-geth/crypto"
)

type EthGetProof struct {
	CommonResponse
	Result AccountResult `json:"result"`
}

// Result structs for GetProof
type AccountResult struct {
	Address      common.Address  `json:"address"`
	AccountProof []string        `json:"accountProof"`
	Balance      *hexutil.Big    `json:"balance"`
	CodeHash     common.Hash     `json:"codeHash"`
	Nonce        hexutil.Uint64  `json:"nonce"`
	StorageHash  common.Hash     `json:"storageHash"`
	StorageProof []StorageResult `json:"storageProof"`
}
type StorageResult struct {
	Key   string       `json:"key"`
	Value *hexutil.Big `json:"value"`
	Proof []string     `json:"proof"`
}

func proofs(url string, block int, account common.Address) {
	var client = &http.Client{
		Timeout: time.Second * 600,
	}
	reqID := 0
	reqID++
	template := `{"jsonrpc":"2.0","method":"eth_getProof","params":["0x%x",[],"0x%x"],"id":%d}`
	var proof EthGetProof
	if err := post(client, url, fmt.Sprintf(template, common.FromHex("0xaCf95bD28Cd5f0A669eA3B9F9cF97Cce7574257c"), 8956072, reqID), &proof); err != nil {
		fmt.Printf("Could not get block number: %v\n", err)
		return
	}
	if proof.Error != nil {
		fmt.Printf("Error retrieving proof: %d %s\n", proof.Error.Code, proof.Error.Message)
		return
	}
	for _, p := range proof.Result.AccountProof {
		b := common.FromHex(p)
		h := crypto.Keccak256(b)
		fmt.Printf("%x: %s\n", h, p)
	}
}
