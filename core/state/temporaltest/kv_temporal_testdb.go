package temporaltest

import (
	"context"
	"testing"

	"github.com/ledgerwatch/erigon-lib/common/datadir"
	"github.com/ledgerwatch/erigon-lib/etconfig2"
	"github.com/ledgerwatch/erigon-lib/kv"
	"github.com/ledgerwatch/erigon-lib/kv/kvcfg"
	"github.com/ledgerwatch/erigon-lib/kv/memdb"
	"github.com/ledgerwatch/erigon-lib/kv/temporal"
	"github.com/ledgerwatch/erigon-lib/state"
	"github.com/ledgerwatch/log/v3"
)

// TODO: need remove `gspec` param (move SystemContractCodeLookup feature somewhere)
func NewTestDB(tb testing.TB, dirs datadir.Dirs) (histV3 bool, db kv.RwDB, agg *state.Aggregator) {
	historyV3 := true
	logger := log.New()

	if tb != nil {
		db = memdb.NewTestDB(tb)
	} else {
		db = memdb.New(dirs.DataDir)
	}
	var err error
	err = db.UpdateNosync(context.Background(), func(tx kv.RwTx) error {
		_, _ = kvcfg.HistoryV3.WriteOnce(tx, historyV3)
		return nil
	})
	if err != nil {
		panic(err)
	}

	agg, err = state.NewAggregator(context.Background(), dirs, etconfig2.HistoryV3AggregationStep, db, logger)
	if err != nil {
		panic(err)
	}
	if err := agg.OpenFolder(false); err != nil {
		panic(err)
	}

	db, err = temporal.New(db, agg)
	if err != nil {
		panic(err)
	}
	return true, db, agg
}
