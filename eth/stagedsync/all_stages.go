package stagedsync

import (
	"fmt"

	"github.com/huandu/xstrings"
	"github.com/prometheus/client_golang/prometheus"

	"github.com/ledgerwatch/erigon-lib/kv"
	"github.com/ledgerwatch/erigon-lib/metrics"
	"github.com/ledgerwatch/erigon/eth/stagedsync/stages"
)

var syncMetrics = map[stages.SyncStage]prometheus.Gauge{}

func init() {
	for _, v := range stages.AllStages {
		syncMetrics[v] = metrics.GetOrCreateGauge(
			fmt.Sprintf(
				`sync{stage="%s"}`,
				xstrings.ToSnakeCase(string(v)),
			),
		)
	}
}

// UpdateMetrics - need update metrics manually because current "metrics" package doesn't support labels
// need to fix it in future
func UpdateMetrics(tx kv.Tx) error {
	for id, m := range syncMetrics {
		progress, err := stages.GetStageProgress(tx, id)
		if err != nil {
			return err
		}
		m.Set(float64(progress))
	}
	return nil
}
