package stagedsync

import (
	"github.com/gateway-fm/cdk-erigon-lib/kv"
)

func PromoteHistory(logPrefix string, tx kv.RwTx, changesetBucket string, start, stop uint64, cfg HistoryCfg, quit <-chan struct{}) error {
	return promoteHistory(logPrefix, tx, changesetBucket, start, stop, cfg, quit)
}