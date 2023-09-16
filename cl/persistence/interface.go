package persistence

import (
	"context"
	"database/sql"
	"io"

	libcommon "github.com/ledgerwatch/erigon-lib/common"
	"github.com/ledgerwatch/erigon/cl/cltypes"
	"github.com/ledgerwatch/erigon/cl/sentinel/peers"
)

type BlockSource interface {
	GetRange(tx *sql.Tx, ctx context.Context, from uint64, count uint64) ([]*peers.PeeredObject[*cltypes.SignedBeaconBlock], error)
	PurgeRange(tx *sql.Tx, ctx context.Context, from uint64, count uint64) error
	ReadBlockToWriter(ctx context.Context, w io.Writer, slot uint64, blockRoot libcommon.Hash) error
}

type BeaconChainWriter interface {
	WriteBlock(tx *sql.Tx, ctx context.Context, block *cltypes.SignedBeaconBlock, canonical bool) error
}

type BeaconChainDatabase interface {
	BlockSource
	BeaconChainWriter
}
