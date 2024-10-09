package tests

import (
	"path/filepath"
	"testing"

	"github.com/ledgerwatch/erigon-lib/config3"
	"github.com/ledgerwatch/log/v3"
)

func TestExecutionSpec(t *testing.T) {
	if config3.EnableHistoryV3InTest {
		t.Skip("fix me in e3 please")
	}

	defer log.Root().SetHandler(log.Root().GetHandler())
	log.Root().SetHandler(log.LvlFilterHandler(log.LvlError, log.StderrHandler))

	bt := new(testMatcher)

	path := filepath.Join(".", "execution-spec-tests", "prague", "eip7702_set_code_tx", "set_code_txs", "set_code_to_sstore.json")

	checkStateRoot := true

	bt.runTestFile(t, path, "", func(t *testing.T, name string, test *BlockTest) {
		// import pre accounts & construct test genesis block & state root
		if err := bt.checkFailure(t, test.Run(t, checkStateRoot)); err != nil {
			t.Error(err)
		}
	})
}
