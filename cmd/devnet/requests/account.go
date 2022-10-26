package requests

import (
	"encoding/json"
	"fmt"
	"github.com/ledgerwatch/erigon/cmd/devnet/models"
	"github.com/ledgerwatch/erigon/cmd/rpctest/rpctest"
	"github.com/ledgerwatch/erigon/common"
	"strconv"
)

func GetBalance(reqId int, address common.Address, blockNum models.BlockNumber) (uint64, error) {
	reqGen := initialiseRequestGenerator(reqId)
	var b rpctest.EthBalance

	reqStr := reqGen.getBalance(address, blockNum)
	if res := reqGen.Erigon(models.ETHGetBalance, reqStr, &b); res.Err != nil {
		return 0, fmt.Errorf("failed to get balance: %v", res.Err)
	}

	bal, err := json.Marshal(b.Balance)
	if err != nil {
		fmt.Println(err)
	}

	balStr := string(bal)[3 : len(bal)-1]
	balance, err := strconv.ParseInt(balStr, 16, 64)
	if err != nil {
		return 0, fmt.Errorf("cannot convert balance to decimal: %v", err)
	}

	return uint64(balance), nil
}
