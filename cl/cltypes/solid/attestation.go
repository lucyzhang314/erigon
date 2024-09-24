package solid

import (
	"encoding/json"

	"github.com/erigontech/erigon-lib/types/ssz"
	"github.com/erigontech/erigon/cl/clparams"
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
	SetAggregationBits(bits []byte)
	SetAttestationData(data AttestationData)
	SetSignature(signature [96]byte)
}

func SSZDecodeAttestation(buf []byte, version clparams.StateVersion) (Attestation, error) {
	switch version {
	case clparams.Phase0Version,
		clparams.AltairVersion,
		clparams.BellatrixVersion,
		clparams.CapellaVersion,
		clparams.DenebVersion:
		att := &AttestationDeneb{}
		if err := att.DecodeSSZ(buf, int(version)); err != nil {
			return nil, err
		}
	case clparams.ElectraVersion:
		att := &AttestationElectra{}
		if err := att.DecodeSSZ(buf, int(version)); err != nil {
			return nil, err
		}
	}
	return nil, nil
}
