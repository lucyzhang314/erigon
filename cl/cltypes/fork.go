package cltypes

import (
	"fmt"

	"github.com/ledgerwatch/erigon-lib/types/ssz"

	"github.com/ledgerwatch/erigon/cl/clparams"
	"github.com/ledgerwatch/erigon/cl/merkle_tree"
	ssz2 "github.com/ledgerwatch/erigon/cl/ssz"
)

// Fork data, contains if we were on bellatrix/alteir/phase0 and transition epoch.
type Fork struct {
	PreviousVersion [4]byte
	CurrentVersion  [4]byte
	Epoch           uint64
}

func (f *Fork) Copy() *Fork {
	return &Fork{
		PreviousVersion: f.PreviousVersion,
		CurrentVersion:  f.CurrentVersion,
		Epoch:           f.Epoch,
	}
}

func (f *Fork) EncodeSSZ(dst []byte) ([]byte, error) {
	return ssz2.Encode(dst, f.PreviousVersion[:], f.CurrentVersion[:], f.Epoch)
}

func (f *Fork) DecodeSSZ(buf []byte, _ int) error {
	if len(buf) < f.EncodingSizeSSZ() {
		return fmt.Errorf("[Fork] err: %s", ssz.ErrLowBufferSize)
	}
	copy(f.PreviousVersion[:], buf)
	copy(f.CurrentVersion[:], buf[clparams.VersionLength:])
	f.Epoch = ssz.UnmarshalUint64SSZ(buf[clparams.VersionLength*2:])
	return nil
}

func (f *Fork) EncodingSizeSSZ() int {
	return clparams.VersionLength*2 + 8
}

func (f *Fork) HashSSZ() ([32]byte, error) {
	return merkle_tree.HashTreeRoot(f.PreviousVersion[:], f.CurrentVersion[:], f.Epoch)
}
