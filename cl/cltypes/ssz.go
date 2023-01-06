package cltypes

import (
	"encoding/binary"

	ssz "github.com/ferranbt/fastssz"
)

type ObjectSSZ interface {
	ssz.Marshaler
	ssz.Unmarshaler

	HashTreeRoot() ([32]byte, error)
}

type EncodableSSZ interface {
	Marshaler
	Unmarshaler
}

type Marshaler interface {
	MarshalSSZ() ([]byte, error)
	SizeSSZ() int
}

type Unmarshaler interface {
	UnmarshalSSZ(buf []byte) error
}

func MarshalUint64SSZ(x uint64) []byte {
	buf := make([]byte, 8)
	binary.LittleEndian.PutUint64(buf, x)
	return buf
}

func UnmarshalUint64SSZ(x []byte) uint64 {
	return binary.LittleEndian.Uint64(x)
}
