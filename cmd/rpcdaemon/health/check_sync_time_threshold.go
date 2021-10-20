package health

import (
	"context"
	"fmt"
	"time"
)

func checkSyncTimeThreshold(syncTimeThreshold uint, api EthAPI) error {
	if api == nil {
		return fmt.Errorf("no connection to the Erigon server or `eth` namespace isn't enabled")
	}
	status, err := api.LastSyncInfo(context.Background())
	if err != nil {
		return err
	}
	lastSyncTime, ok := status["lastSyncTime"].(uint64)
	if !ok {
		return fmt.Errorf("invalid last sync time")
	}
	tm := time.Unix(int64(lastSyncTime), 0)
	if time.Since(tm).Seconds() > float64(syncTimeThreshold) {
		return fmt.Errorf("time from the last sync (%v) exceed (%v seconds)", tm.Unix(), syncTimeThreshold)
	}

	return nil
}
