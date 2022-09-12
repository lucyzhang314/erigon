package main

import (
	"time"

	"github.com/ledgerwatch/erigon-lib/kv/mdbx"
	"github.com/ledgerwatch/log/v3"
)

/*func readAccountKey(tx kv.RwTx, address []byte) ([]byte, error) {
	return tx.GetOne(PedersenHashedAccountsLookup, address)
}

func readStorageKey(tx kv.RwTx, address []byte, storageKey []byte) ([]byte, error) {
	return tx.GetOne(PedersenHashedStorageLookup, append(address, storageKey...))
}

func readCodeKey(tx kv.RwTx, address []byte, index *uint256.Int) ([]byte, error) {
	lookupKey := make([]byte, 24)
	copy(lookupKey, address)
	binary.BigEndian.PutUint32(lookupKey[20:], uint32(index.Uint64()))
	return tx.GetOne(PedersenHashedCodeLookup, lookupKey)
}*/

func IncrementVerkleTree(cfg optionsCfg) error {
	start := time.Now()
	db, err := mdbx.Open(cfg.stateDb, log.Root(), true)
	if err != nil {
		log.Error("Error while opening database", "err", err.Error())
		return err
	}
	defer db.Close()

	vDb, err := mdbx.Open(cfg.verkleDb, log.Root(), false)
	if err != nil {
		log.Error("Error while opening db transaction", "err", err.Error())
		return err
	}
	defer vDb.Close()

	vTx, err := vDb.BeginRw(cfg.ctx)
	if err != nil {
		return err
	}
	defer vTx.Rollback()

	tx, err := db.BeginRo(cfg.ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if err := initDB(vTx); err != nil {
		return err
	}
	if err := incrementAccount(vTx, tx, cfg); err != nil {
		return err
	}
	if err := incrementStorage(vTx, tx, cfg); err != nil {
		return err
	}
	log.Info("Finished", "elapesed", time.Since(start))
	return vTx.Commit()
}
