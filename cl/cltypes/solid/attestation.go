package solid

import (
	"encoding/json"

	"github.com/erigontech/erigon-lib/types/ssz"
	ssz2 "github.com/erigontech/erigon/cl/ssz"
)

type Attestation interface {
	json.Marshaler
	json.Unmarshaler
	ssz2.SizedObjectSSZ
	ssz.HashableSSZ
	Static() bool
	Copy() Attestation
	AggregationBits() []byte
	AttestantionData() AttestationData
	Signature() (o [96]byte)
}
