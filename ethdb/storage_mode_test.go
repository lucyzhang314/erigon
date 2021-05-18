package ethdb

import (
	"reflect"
	"testing"

	"github.com/davecgh/go-spew/spew"
)

func TestSetStorageModeIfNotExist(t *testing.T) {
	db := NewTestDB(t)
	sm, err := GetStorageModeFromDB(db)
	if err != nil {
		t.Fatal(err)
	}

	if !reflect.DeepEqual(sm, StorageMode{Initialised: true}) {
		t.Fatal()
	}

	err = SetStorageModeIfNotExist(db, StorageMode{
		true,
		true,
		true,
		true,
		true,
		true,
	})
	if err != nil {
		t.Fatal(err)
	}

	sm, err = GetStorageModeFromDB(db)
	if err != nil {
		t.Fatal(err)
	}

	if !reflect.DeepEqual(sm, StorageMode{
		true,
		true,
		true,
		true,
		true,
		true,
	}) {
		spew.Dump(sm)
		t.Fatal("not equal")
	}
}
