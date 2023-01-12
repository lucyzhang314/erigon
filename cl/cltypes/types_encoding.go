// Code generated by fastssz. DO NOT EDIT.
// Hash: 11844f06c664f9ff5e7ea9532e9c99f98dd5c9c881933713e893ffb0466b5772
package cltypes

import (
	"github.com/ledgerwatch/erigon/cl/clparams"
	"github.com/ledgerwatch/erigon/core/types"
	ssz "github.com/prysmaticlabs/fastssz"
)

// MarshalSSZ ssz marshals the SyncCommittee object
func (s *SyncCommittee) MarshalSSZ() ([]byte, error) {
	return ssz.MarshalSSZ(s)
}

// MarshalSSZTo ssz marshals the SyncCommittee object to a target array
func (s *SyncCommittee) MarshalSSZTo(buf []byte) (dst []byte, err error) {
	dst = buf

	// Field (0) 'PubKeys'
	if size := len(s.PubKeys); size != 512 {
		err = ssz.ErrVectorLengthFn("--.PubKeys", size, 512)
		return
	}
	for ii := 0; ii < 512; ii++ {
		dst = append(dst, s.PubKeys[ii][:]...)
	}

	// Field (1) 'AggregatePublicKey'
	dst = append(dst, s.AggregatePublicKey[:]...)

	return
}

// UnmarshalSSZ ssz unmarshals the SyncCommittee object
func (s *SyncCommittee) UnmarshalSSZ(buf []byte) error {
	var err error
	size := uint64(len(buf))
	if size != 24624 {
		return ssz.ErrSize
	}

	// Field (0) 'PubKeys'
	s.PubKeys = make([][48]byte, 512)
	for ii := 0; ii < 512; ii++ {
		copy(s.PubKeys[ii][:], buf[0:24576][ii*48:(ii+1)*48])
	}

	// Field (1) 'AggregatePublicKey'
	copy(s.AggregatePublicKey[:], buf[24576:24624])

	return err
}

// SizeSSZ returns the ssz encoded size in bytes for the SyncCommittee object
func (s *SyncCommittee) SizeSSZ() (size int) {
	size = 24624
	return
}

// HashTreeRoot ssz hashes the SyncCommittee object
func (s *SyncCommittee) HashTreeRoot() ([32]byte, error) {
	return ssz.HashWithDefaultHasher(s)
}

// HashTreeRootWith ssz hashes the SyncCommittee object with a hasher
func (s *SyncCommittee) HashTreeRootWith(hh *ssz.Hasher) (err error) {
	indx := hh.Index()

	// Field (0) 'PubKeys'
	{
		if size := len(s.PubKeys); size != 512 {
			err = ssz.ErrVectorLengthFn("--.PubKeys", size, 512)
			return
		}
		subIndx := hh.Index()
		for _, i := range s.PubKeys {
			hh.PutBytes(i[:])
		}

		if ssz.EnableVectorizedHTR {
			hh.MerkleizeVectorizedHTR(subIndx)
		} else {
			hh.Merkleize(subIndx)
		}
	}

	// Field (1) 'AggregatePublicKey'
	hh.PutBytes(s.AggregatePublicKey[:])

	if ssz.EnableVectorizedHTR {
		hh.MerkleizeVectorizedHTR(indx)
	} else {
		hh.Merkleize(indx)
	}
	return
}

// MarshalSSZ ssz marshals the LightClientBootstrap object
func (l *LightClientBootstrap) MarshalSSZ() ([]byte, error) {
	return ssz.MarshalSSZ(l)
}

// MarshalSSZTo ssz marshals the LightClientBootstrap object to a target array
func (l *LightClientBootstrap) MarshalSSZTo(buf []byte) (dst []byte, err error) {
	dst = buf

	// Field (0) 'Header'
	if l.Header == nil {
		l.Header = new(BeaconBlockHeader)
	}
	dst = l.Header.EncodeSSZ(dst)
	// Field (1) 'CurrentSyncCommittee'
	if l.CurrentSyncCommittee == nil {
		l.CurrentSyncCommittee = new(SyncCommittee)
	}
	if dst, err = l.CurrentSyncCommittee.MarshalSSZTo(dst); err != nil {
		return
	}

	// Field (2) 'CurrentSyncCommitteeBranch'
	if size := len(l.CurrentSyncCommitteeBranch); size != 5 {
		err = ssz.ErrVectorLengthFn("--.CurrentSyncCommitteeBranch", size, 5)
		return
	}
	for ii := 0; ii < 5; ii++ {
		if size := len(l.CurrentSyncCommitteeBranch[ii]); size != 32 {
			err = ssz.ErrBytesLengthFn("--.CurrentSyncCommitteeBranch[ii]", size, 32)
			return
		}
		dst = append(dst, l.CurrentSyncCommitteeBranch[ii]...)
	}

	return
}

// UnmarshalSSZ ssz unmarshals the LightClientBootstrap object
func (l *LightClientBootstrap) UnmarshalSSZ(buf []byte) error {
	var err error
	size := uint64(len(buf))
	if size != 24896 {
		return ssz.ErrSize
	}

	// Field (0) 'Header'
	if l.Header == nil {
		l.Header = new(BeaconBlockHeader)
	}
	if err = l.Header.DecodeSSZ(buf[0:112]); err != nil {
		return err
	}

	// Field (1) 'CurrentSyncCommittee'
	if l.CurrentSyncCommittee == nil {
		l.CurrentSyncCommittee = new(SyncCommittee)
	}
	if err = l.CurrentSyncCommittee.UnmarshalSSZ(buf[112:24736]); err != nil {
		return err
	}

	// Field (2) 'CurrentSyncCommitteeBranch'
	l.CurrentSyncCommitteeBranch = make([][]byte, 5)
	for ii := 0; ii < 5; ii++ {
		if cap(l.CurrentSyncCommitteeBranch[ii]) == 0 {
			l.CurrentSyncCommitteeBranch[ii] = make([]byte, 0, len(buf[24736:24896][ii*32:(ii+1)*32]))
		}
		l.CurrentSyncCommitteeBranch[ii] = append(l.CurrentSyncCommitteeBranch[ii], buf[24736:24896][ii*32:(ii+1)*32]...)
	}

	return err
}

// SizeSSZ returns the ssz encoded size in bytes for the LightClientBootstrap object
func (l *LightClientBootstrap) SizeSSZ() (size int) {
	size = 24896
	return
}

// MarshalSSZ ssz marshals the LightClientUpdate object
func (l *LightClientUpdate) MarshalSSZ() ([]byte, error) {
	return ssz.MarshalSSZ(l)
}

// MarshalSSZTo ssz marshals the LightClientUpdate object to a target array
func (l *LightClientUpdate) MarshalSSZTo(buf []byte) (dst []byte, err error) {
	dst = buf

	// Field (0) 'AttestedHeader'
	if l.AttestedHeader == nil {
		l.AttestedHeader = new(BeaconBlockHeader)
	}
	dst = l.AttestedHeader.EncodeSSZ(dst)
	// Field (1) 'NextSyncCommitee'
	if l.NextSyncCommitee == nil {
		l.NextSyncCommitee = new(SyncCommittee)
	}
	if dst, err = l.NextSyncCommitee.MarshalSSZTo(dst); err != nil {
		return
	}

	// Field (2) 'NextSyncCommitteeBranch'
	if size := len(l.NextSyncCommitteeBranch); size != 5 {
		err = ssz.ErrVectorLengthFn("--.NextSyncCommitteeBranch", size, 5)
		return
	}
	for ii := 0; ii < 5; ii++ {
		if size := len(l.NextSyncCommitteeBranch[ii]); size != 32 {
			err = ssz.ErrBytesLengthFn("--.NextSyncCommitteeBranch[ii]", size, 32)
			return
		}
		dst = append(dst, l.NextSyncCommitteeBranch[ii]...)
	}

	// Field (3) 'FinalizedHeader'
	if l.FinalizedHeader == nil {
		l.FinalizedHeader = new(BeaconBlockHeader)
	}
	dst = l.FinalizedHeader.EncodeSSZ(dst)

	// Field (4) 'FinalityBranch'
	if size := len(l.FinalityBranch); size != 6 {
		err = ssz.ErrVectorLengthFn("--.FinalityBranch", size, 6)
		return
	}
	for ii := 0; ii < 6; ii++ {
		if size := len(l.FinalityBranch[ii]); size != 32 {
			err = ssz.ErrBytesLengthFn("--.FinalityBranch[ii]", size, 32)
			return
		}
		dst = append(dst, l.FinalityBranch[ii]...)
	}

	// Field (5) 'SyncAggregate'
	if l.SyncAggregate == nil {
		l.SyncAggregate = new(SyncAggregate)
	}
	dst = l.SyncAggregate.EncodeSSZ(dst)

	// Field (6) 'SignatureSlot'
	dst = ssz.MarshalUint64(dst, l.SignatureSlot)

	return
}

// UnmarshalSSZ ssz unmarshals the LightClientUpdate object
func (l *LightClientUpdate) UnmarshalSSZ(buf []byte) error {
	var err error
	size := uint64(len(buf))
	if size != 25368 {
		return ssz.ErrSize
	}

	// Field (0) 'AttestedHeader'
	if l.AttestedHeader == nil {
		l.AttestedHeader = new(BeaconBlockHeader)
	}
	if err = l.AttestedHeader.DecodeSSZ(buf[0:112]); err != nil {
		return err
	}

	// Field (1) 'NextSyncCommitee'
	if l.NextSyncCommitee == nil {
		l.NextSyncCommitee = new(SyncCommittee)
	}
	if err = l.NextSyncCommitee.UnmarshalSSZ(buf[112:24736]); err != nil {
		return err
	}

	// Field (2) 'NextSyncCommitteeBranch'
	l.NextSyncCommitteeBranch = make([][]byte, 5)
	for ii := 0; ii < 5; ii++ {
		if cap(l.NextSyncCommitteeBranch[ii]) == 0 {
			l.NextSyncCommitteeBranch[ii] = make([]byte, 0, len(buf[24736:24896][ii*32:(ii+1)*32]))
		}
		l.NextSyncCommitteeBranch[ii] = append(l.NextSyncCommitteeBranch[ii], buf[24736:24896][ii*32:(ii+1)*32]...)
	}

	// Field (3) 'FinalizedHeader'
	if l.FinalizedHeader == nil {
		l.FinalizedHeader = new(BeaconBlockHeader)
	}
	if err = l.FinalizedHeader.DecodeSSZ(buf[24896:25008]); err != nil {
		return err
	}

	// Field (4) 'FinalityBranch'
	l.FinalityBranch = make([][]byte, 6)
	for ii := 0; ii < 6; ii++ {
		if cap(l.FinalityBranch[ii]) == 0 {
			l.FinalityBranch[ii] = make([]byte, 0, len(buf[25008:25200][ii*32:(ii+1)*32]))
		}
		l.FinalityBranch[ii] = append(l.FinalityBranch[ii], buf[25008:25200][ii*32:(ii+1)*32]...)
	}

	// Field (5) 'SyncAggregate'
	if l.SyncAggregate == nil {
		l.SyncAggregate = new(SyncAggregate)
	}
	if err = l.SyncAggregate.DecodeSSZ(buf[25200:25360]); err != nil {
		return err
	}

	// Field (6) 'SignatureSlot'
	l.SignatureSlot = ssz.UnmarshallUint64(buf[25360:25368])

	return err
}

// SizeSSZ returns the ssz encoded size in bytes for the LightClientUpdate object
func (l *LightClientUpdate) SizeSSZ() (size int) {
	size = 25368
	return
}

// MarshalSSZ ssz marshals the LightClientFinalityUpdate object
func (l *LightClientFinalityUpdate) MarshalSSZ() ([]byte, error) {
	return ssz.MarshalSSZ(l)
}

// MarshalSSZTo ssz marshals the LightClientFinalityUpdate object to a target array
func (l *LightClientFinalityUpdate) MarshalSSZTo(buf []byte) (dst []byte, err error) {
	dst = buf

	// Field (0) 'AttestedHeader'
	if l.AttestedHeader == nil {
		l.AttestedHeader = new(BeaconBlockHeader)
	}
	dst =  l.AttestedHeader.EncodeSSZ(dst)

	// Field (1) 'FinalizedHeader'
	if l.FinalizedHeader == nil {
		l.FinalizedHeader = new(BeaconBlockHeader)
	}
	dst = l.FinalizedHeader.EncodeSSZ(dst)
	// Field (2) 'FinalityBranch'
	if size := len(l.FinalityBranch); size != 6 {
		err = ssz.ErrVectorLengthFn("--.FinalityBranch", size, 6)
		return
	}
	for ii := 0; ii < 6; ii++ {
		if size := len(l.FinalityBranch[ii]); size != 32 {
			err = ssz.ErrBytesLengthFn("--.FinalityBranch[ii]", size, 32)
			return
		}
		dst = append(dst, l.FinalityBranch[ii]...)
	}

	// Field (3) 'SyncAggregate'
	if l.SyncAggregate == nil {
		l.SyncAggregate = new(SyncAggregate)
	}
	dst = l.SyncAggregate.EncodeSSZ(dst)
	

	// Field (4) 'SignatureSlot'
	dst = ssz.MarshalUint64(dst, l.SignatureSlot)

	return
}

// UnmarshalSSZ ssz unmarshals the LightClientFinalityUpdate object
func (l *LightClientFinalityUpdate) UnmarshalSSZ(buf []byte) error {
	var err error
	size := uint64(len(buf))
	if size != 584 {
		return ssz.ErrSize
	}

	// Field (0) 'AttestedHeader'
	if l.AttestedHeader == nil {
		l.AttestedHeader = new(BeaconBlockHeader)
	}
	if err = l.AttestedHeader.DecodeSSZ(buf[0:112]); err != nil {
		return err
	}

	// Field (1) 'FinalizedHeader'
	if l.FinalizedHeader == nil {
		l.FinalizedHeader = new(BeaconBlockHeader)
	}
	if err = l.FinalizedHeader.DecodeSSZ(buf[112:224]); err != nil {
		return err
	}

	// Field (2) 'FinalityBranch'
	l.FinalityBranch = make([][]byte, 6)
	for ii := 0; ii < 6; ii++ {
		if cap(l.FinalityBranch[ii]) == 0 {
			l.FinalityBranch[ii] = make([]byte, 0, len(buf[224:416][ii*32:(ii+1)*32]))
		}
		l.FinalityBranch[ii] = append(l.FinalityBranch[ii], buf[224:416][ii*32:(ii+1)*32]...)
	}

	// Field (3) 'SyncAggregate'
	if l.SyncAggregate == nil {
		l.SyncAggregate = new(SyncAggregate)
	}
	if err = l.SyncAggregate.DecodeSSZ(buf[416:576]); err != nil {
		return err
	}

	// Field (4) 'SignatureSlot'
	l.SignatureSlot = ssz.UnmarshallUint64(buf[576:584])

	return err
}

// SizeSSZ returns the ssz encoded size in bytes for the LightClientFinalityUpdate object
func (l *LightClientFinalityUpdate) SizeSSZ() (size int) {
	size = 584
	return
}

// MarshalSSZ ssz marshals the LightClientOptimisticUpdate object
func (l *LightClientOptimisticUpdate) MarshalSSZ() ([]byte, error) {
	return ssz.MarshalSSZ(l)
}

// MarshalSSZTo ssz marshals the LightClientOptimisticUpdate object to a target array
func (l *LightClientOptimisticUpdate) MarshalSSZTo(buf []byte) (dst []byte, err error) {
	dst = buf

	// Field (0) 'AttestedHeader'
	if l.AttestedHeader == nil {
		l.AttestedHeader = new(BeaconBlockHeader)
	}
	dst = l.AttestedHeader.EncodeSSZ(dst)
	// Field (1) 'SyncAggregate'
	if l.SyncAggregate == nil {
		l.SyncAggregate = new(SyncAggregate)
	}
	dst = l.SyncAggregate.EncodeSSZ(dst)

	// Field (2) 'SignatureSlot'
	dst = ssz.MarshalUint64(dst, l.SignatureSlot)

	return
}

// UnmarshalSSZ ssz unmarshals the LightClientOptimisticUpdate object
func (l *LightClientOptimisticUpdate) UnmarshalSSZ(buf []byte) error {
	var err error
	size := uint64(len(buf))
	if size != 280 {
		return ssz.ErrSize
	}

	// Field (0) 'AttestedHeader'
	if l.AttestedHeader == nil {
		l.AttestedHeader = new(BeaconBlockHeader)
	}
	if err = l.AttestedHeader.DecodeSSZ(buf[0:112]); err != nil {
		return err
	}

	// Field (1) 'SyncAggregate'
	if l.SyncAggregate == nil {
		l.SyncAggregate = new(SyncAggregate)
	}
	if err = l.SyncAggregate.DecodeSSZ(buf[112:272]); err != nil {
		return err
	}

	// Field (2) 'SignatureSlot'
	l.SignatureSlot = ssz.UnmarshallUint64(buf[272:280])

	return err
}

// SizeSSZ returns the ssz encoded size in bytes for the LightClientOptimisticUpdate object
func (l *LightClientOptimisticUpdate) SizeSSZ() (size int) {
	size = 280
	return
}


// MarshalSSZ ssz marshals the Fork object
func (f *Fork) MarshalSSZ() ([]byte, error) {
	return ssz.MarshalSSZ(f)
}

// MarshalSSZTo ssz marshals the Fork object to a target array
func (f *Fork) MarshalSSZTo(buf []byte) (dst []byte, err error) {
	dst = buf

	// Field (0) 'PreviousVersion'
	dst = append(dst, f.PreviousVersion[:]...)

	// Field (1) 'CurrentVersion'
	dst = append(dst, f.CurrentVersion[:]...)

	// Field (2) 'Epoch'
	dst = ssz.MarshalUint64(dst, f.Epoch)

	return
}

// UnmarshalSSZ ssz unmarshals the Fork object
func (f *Fork) UnmarshalSSZ(buf []byte) error {
	var err error
	size := uint64(len(buf))
	if size != 16 {
		return ssz.ErrSize
	}

	// Field (0) 'PreviousVersion'
	copy(f.PreviousVersion[:], buf[0:4])

	// Field (1) 'CurrentVersion'
	copy(f.CurrentVersion[:], buf[4:8])

	// Field (2) 'Epoch'
	f.Epoch = ssz.UnmarshallUint64(buf[8:16])

	return err
}

// SizeSSZ returns the ssz encoded size in bytes for the Fork object
func (f *Fork) SizeSSZ() (size int) {
	size = 16
	return
}

// HashTreeRoot ssz hashes the Fork object
func (f *Fork) HashTreeRoot() ([32]byte, error) {
	return ssz.HashWithDefaultHasher(f)
}

// HashTreeRootWith ssz hashes the Fork object with a hasher
func (f *Fork) HashTreeRootWith(hh *ssz.Hasher) (err error) {
	indx := hh.Index()

	// Field (0) 'PreviousVersion'
	hh.PutBytes(f.PreviousVersion[:])

	// Field (1) 'CurrentVersion'
	hh.PutBytes(f.CurrentVersion[:])

	// Field (2) 'Epoch'
	hh.PutUint64(f.Epoch)

	if ssz.EnableVectorizedHTR {
		hh.MerkleizeVectorizedHTR(indx)
	} else {
		hh.Merkleize(indx)
	}
	return
}

// MarshalSSZ ssz marshals the Validator object
func (v *Validator) MarshalSSZ() ([]byte, error) {
	return ssz.MarshalSSZ(v)
}

// MarshalSSZTo ssz marshals the Validator object to a target array
func (v *Validator) MarshalSSZTo(buf []byte) (dst []byte, err error) {
	dst = buf

	// Field (0) 'PublicKey'
	dst = append(dst, v.PublicKey[:]...)

	// Field (1) 'WithdrawalCredentials'
	if size := len(v.WithdrawalCredentials); size != 32 {
		err = ssz.ErrBytesLengthFn("--.WithdrawalCredentials", size, 32)
		return
	}
	dst = append(dst, v.WithdrawalCredentials...)

	// Field (2) 'EffectiveBalance'
	dst = ssz.MarshalUint64(dst, v.EffectiveBalance)

	// Field (3) 'Slashed'
	dst = ssz.MarshalBool(dst, v.Slashed)

	// Field (4) 'ActivationEligibilityEpoch'
	dst = ssz.MarshalUint64(dst, v.ActivationEligibilityEpoch)

	// Field (5) 'ActivationEpoch'
	dst = ssz.MarshalUint64(dst, v.ActivationEpoch)

	// Field (6) 'ExitEpoch'
	dst = ssz.MarshalUint64(dst, v.ExitEpoch)

	// Field (7) 'WithdrawableEpoch'
	dst = ssz.MarshalUint64(dst, v.WithdrawableEpoch)

	return
}

// UnmarshalSSZ ssz unmarshals the Validator object
func (v *Validator) UnmarshalSSZ(buf []byte) error {
	var err error
	size := uint64(len(buf))
	if size != 121 {
		return ssz.ErrSize
	}

	// Field (0) 'PublicKey'
	copy(v.PublicKey[:], buf[0:48])

	// Field (1) 'WithdrawalCredentials'
	if cap(v.WithdrawalCredentials) == 0 {
		v.WithdrawalCredentials = make([]byte, 0, len(buf[48:80]))
	}
	v.WithdrawalCredentials = append(v.WithdrawalCredentials, buf[48:80]...)

	// Field (2) 'EffectiveBalance'
	v.EffectiveBalance = ssz.UnmarshallUint64(buf[80:88])

	// Field (3) 'Slashed'
	v.Slashed = ssz.UnmarshalBool(buf[88:89])

	// Field (4) 'ActivationEligibilityEpoch'
	v.ActivationEligibilityEpoch = ssz.UnmarshallUint64(buf[89:97])

	// Field (5) 'ActivationEpoch'
	v.ActivationEpoch = ssz.UnmarshallUint64(buf[97:105])

	// Field (6) 'ExitEpoch'
	v.ExitEpoch = ssz.UnmarshallUint64(buf[105:113])

	// Field (7) 'WithdrawableEpoch'
	v.WithdrawableEpoch = ssz.UnmarshallUint64(buf[113:121])

	return err
}

// SizeSSZ returns the ssz encoded size in bytes for the Validator object
func (v *Validator) SizeSSZ() (size int) {
	size = 121
	return
}

// HashTreeRoot ssz hashes the Validator object
func (v *Validator) HashTreeRoot() ([32]byte, error) {
	return ssz.HashWithDefaultHasher(v)
}

// HashTreeRootWith ssz hashes the Validator object with a hasher
func (v *Validator) HashTreeRootWith(hh *ssz.Hasher) (err error) {
	indx := hh.Index()

	// Field (0) 'PublicKey'
	hh.PutBytes(v.PublicKey[:])

	// Field (1) 'WithdrawalCredentials'
	if size := len(v.WithdrawalCredentials); size != 32 {
		err = ssz.ErrBytesLengthFn("--.WithdrawalCredentials", size, 32)
		return
	}
	hh.PutBytes(v.WithdrawalCredentials)

	// Field (2) 'EffectiveBalance'
	hh.PutUint64(v.EffectiveBalance)

	// Field (3) 'Slashed'
	hh.PutBool(v.Slashed)

	// Field (4) 'ActivationEligibilityEpoch'
	hh.PutUint64(v.ActivationEligibilityEpoch)

	// Field (5) 'ActivationEpoch'
	hh.PutUint64(v.ActivationEpoch)

	// Field (6) 'ExitEpoch'
	hh.PutUint64(v.ExitEpoch)

	// Field (7) 'WithdrawableEpoch'
	hh.PutUint64(v.WithdrawableEpoch)

	if ssz.EnableVectorizedHTR {
		hh.MerkleizeVectorizedHTR(indx)
	} else {
		hh.Merkleize(indx)
	}
	return
}

// MarshalSSZ ssz marshals the BeaconStateBellatrix object
func (b *BeaconStateBellatrix) MarshalSSZ() ([]byte, error) {
	return ssz.MarshalSSZ(b)
}

// MarshalSSZTo ssz marshals the BeaconStateBellatrix object to a target array
func (b *BeaconStateBellatrix) MarshalSSZTo(buf []byte) (dst []byte, err error) {
	dst = buf
	offset := int(2736633)

	// Field (0) 'GenesisTime'
	dst = ssz.MarshalUint64(dst, b.GenesisTime)

	// Field (1) 'GenesisValidatorsRoot'
	dst = append(dst, b.GenesisValidatorsRoot[:]...)

	// Field (2) 'Slot'
	dst = ssz.MarshalUint64(dst, b.Slot)

	// Field (3) 'Fork'
	if b.Fork == nil {
		b.Fork = new(Fork)
	}
	if dst, err = b.Fork.MarshalSSZTo(dst); err != nil {
		return
	}

	// Field (4) 'LatestBlockHeader'
	if b.LatestBlockHeader == nil {
		b.LatestBlockHeader = new(BeaconBlockHeader)
	}
	dst = b.LatestBlockHeader.EncodeSSZ(dst)

	// Field (5) 'BlockRoots'
	if size := len(b.BlockRoots); size != 8192 {
		err = ssz.ErrVectorLengthFn("--.BlockRoots", size, 8192)
		return
	}
	for ii := 0; ii < 8192; ii++ {
		dst = append(dst, b.BlockRoots[ii][:]...)
	}

	// Field (6) 'StateRoots'
	if size := len(b.StateRoots); size != 8192 {
		err = ssz.ErrVectorLengthFn("--.StateRoots", size, 8192)
		return
	}
	for ii := 0; ii < 8192; ii++ {
		dst = append(dst, b.StateRoots[ii][:]...)
	}

	// Offset (7) 'HistoricalRoots'
	dst = ssz.WriteOffset(dst, offset)
	offset += len(b.HistoricalRoots) * 32

	// Field (8) 'Eth1Data'
	if b.Eth1Data == nil {
		b.Eth1Data = new(Eth1Data)
	}
	dst = b.Eth1Data.EncodeSSZ(dst)

	// Offset (9) 'Eth1DataVotes'
	dst = ssz.WriteOffset(dst, offset)
	offset += len(b.Eth1DataVotes) * 72

	// Field (10) 'Eth1DepositIndex'
	dst = ssz.MarshalUint64(dst, b.Eth1DepositIndex)

	// Offset (11) 'Validators'
	dst = ssz.WriteOffset(dst, offset)
	offset += len(b.Validators) * 121

	// Offset (12) 'Balances'
	dst = ssz.WriteOffset(dst, offset)
	offset += len(b.Balances) * 8

	// Field (13) 'RandaoMixes'
	if size := len(b.RandaoMixes); size != 65536 {
		err = ssz.ErrVectorLengthFn("--.RandaoMixes", size, 65536)
		return
	}
	for ii := 0; ii < 65536; ii++ {
		dst = append(dst, b.RandaoMixes[ii][:]...)
	}

	// Field (14) 'Slashings'
	if size := len(b.Slashings); size != 8192 {
		err = ssz.ErrVectorLengthFn("--.Slashings", size, 8192)
		return
	}
	for ii := 0; ii < 8192; ii++ {
		dst = ssz.MarshalUint64(dst, b.Slashings[ii])
	}

	// Offset (15) 'PreviousEpochParticipation'
	dst = ssz.WriteOffset(dst, offset)
	offset += len(b.PreviousEpochParticipation)

	// Offset (16) 'CurrentEpochParticipation'
	dst = ssz.WriteOffset(dst, offset)
	offset += len(b.CurrentEpochParticipation)

	// Field (17) 'JustificationBits'
	if size := len(b.JustificationBits); size != 1 {
		err = ssz.ErrBytesLengthFn("--.JustificationBits", size, 1)
		return
	}
	dst = append(dst, b.JustificationBits...)

	// Field (18) 'PreviousJustifiedCheckpoint'
	if b.PreviousJustifiedCheckpoint == nil {
		b.PreviousJustifiedCheckpoint = new(Checkpoint)
	}
	 dst = b.PreviousJustifiedCheckpoint.EncodeSSZ(dst)

	// Field (19) 'CurrentJustifiedCheckpoint'
	if b.CurrentJustifiedCheckpoint == nil {
		b.CurrentJustifiedCheckpoint = new(Checkpoint)
	}
	dst = b.CurrentJustifiedCheckpoint.EncodeSSZ(dst);
	// Field (20) 'FinalizedCheckpoint'
	if b.FinalizedCheckpoint == nil {
		b.FinalizedCheckpoint = new(Checkpoint)
	}
	dst = b.FinalizedCheckpoint.EncodeSSZ(dst)

	// Offset (21) 'InactivityScores'
	dst = ssz.WriteOffset(dst, offset)
	offset += len(b.InactivityScores) * 8

	// Field (22) 'CurrentSyncCommittee'
	if b.CurrentSyncCommittee == nil {
		b.CurrentSyncCommittee = new(SyncCommittee)
	}
	if dst, err = b.CurrentSyncCommittee.MarshalSSZTo(dst); err != nil {
		return
	}

	// Field (23) 'NextSyncCommittee'
	if b.NextSyncCommittee == nil {
		b.NextSyncCommittee = new(SyncCommittee)
	}
	if dst, err = b.NextSyncCommittee.MarshalSSZTo(dst); err != nil {
		return
	}

	// Offset (24) 'LatestExecutionPayloadHeader'
	dst = ssz.WriteOffset(dst, offset)
	if b.LatestExecutionPayloadHeader == nil {
		b.LatestExecutionPayloadHeader = new(types.Header)
	}
	offset += b.LatestExecutionPayloadHeader.EncodingSizeSSZ(clparams.BellatrixVersion)

	// Field (7) 'HistoricalRoots'
	if size := len(b.HistoricalRoots); size > 16777216 {
		err = ssz.ErrListTooBigFn("--.HistoricalRoots", size, 16777216)
		return
	}
	for ii := 0; ii < len(b.HistoricalRoots); ii++ {
		dst = append(dst, b.HistoricalRoots[ii][:]...)
	}

	// Field (9) 'Eth1DataVotes'
	if size := len(b.Eth1DataVotes); size > 2048 {
		err = ssz.ErrListTooBigFn("--.Eth1DataVotes", size, 2048)
		return
	}
	for ii := 0; ii < len(b.Eth1DataVotes); ii++ {
		dst = b.Eth1DataVotes[ii].EncodeSSZ(dst)
	}

	// Field (11) 'Validators'
	if size := len(b.Validators); size > 1099511627776 {
		err = ssz.ErrListTooBigFn("--.Validators", size, 1099511627776)
		return
	}
	for ii := 0; ii < len(b.Validators); ii++ {
		if dst, err = b.Validators[ii].MarshalSSZTo(dst); err != nil {
			return
		}
	}

	// Field (12) 'Balances'
	if size := len(b.Balances); size > 1099511627776 {
		err = ssz.ErrListTooBigFn("--.Balances", size, 1099511627776)
		return
	}
	for ii := 0; ii < len(b.Balances); ii++ {
		dst = ssz.MarshalUint64(dst, b.Balances[ii])
	}

	// Field (15) 'PreviousEpochParticipation'
	if size := len(b.PreviousEpochParticipation); size > 1099511627776 {
		err = ssz.ErrBytesLengthFn("--.PreviousEpochParticipation", size, 1099511627776)
		return
	}
	dst = append(dst, b.PreviousEpochParticipation...)

	// Field (16) 'CurrentEpochParticipation'
	if size := len(b.CurrentEpochParticipation); size > 1099511627776 {
		err = ssz.ErrBytesLengthFn("--.CurrentEpochParticipation", size, 1099511627776)
		return
	}
	dst = append(dst, b.CurrentEpochParticipation...)

	// Field (21) 'InactivityScores'
	if size := len(b.InactivityScores); size > 1099511627776 {
		err = ssz.ErrListTooBigFn("--.InactivityScores", size, 1099511627776)
		return
	}
	for ii := 0; ii < len(b.InactivityScores); ii++ {
		dst = ssz.MarshalUint64(dst, b.InactivityScores[ii])
	}
	dst = b.LatestExecutionPayloadHeader.EncodeSSZ(dst)

	return
}

// UnmarshalSSZ ssz unmarshals the BeaconStateBellatrix object
func (b *BeaconStateBellatrix) UnmarshalSSZ(buf []byte) error {
	var err error
	size := uint64(len(buf))
	if size < 2736633 {
		return ssz.ErrSize
	}

	tail := buf
	var o7, o9, o11, o12, o15, o16, o21, o24 uint64

	// Field (0) 'GenesisTime'
	b.GenesisTime = ssz.UnmarshallUint64(buf[0:8])

	// Field (1) 'GenesisValidatorsRoot'
	copy(b.GenesisValidatorsRoot[:], buf[8:40])

	// Field (2) 'Slot'
	b.Slot = ssz.UnmarshallUint64(buf[40:48])

	// Field (3) 'Fork'
	if b.Fork == nil {
		b.Fork = new(Fork)
	}
	if err = b.Fork.UnmarshalSSZ(buf[48:64]); err != nil {
		return err
	}

	// Field (4) 'LatestBlockHeader'
	if b.LatestBlockHeader == nil {
		b.LatestBlockHeader = new(BeaconBlockHeader)
	}
	if err = b.LatestBlockHeader.DecodeSSZ(buf[64:176]); err != nil {
		return err
	}

	// Field (5) 'BlockRoots'
	b.BlockRoots = make([][32]byte, 8192)
	for ii := 0; ii < 8192; ii++ {
		copy(b.BlockRoots[ii][:], buf[176:262320][ii*32:(ii+1)*32])
	}

	// Field (6) 'StateRoots'
	b.StateRoots = make([][32]byte, 8192)
	for ii := 0; ii < 8192; ii++ {
		copy(b.StateRoots[ii][:], buf[262320:524464][ii*32:(ii+1)*32])
	}

	// Offset (7) 'HistoricalRoots'
	if o7 = ssz.ReadOffset(buf[524464:524468]); o7 > size {
		return ssz.ErrOffset
	}

	if o7 < 2736633 {
		return ssz.ErrInvalidVariableOffset
	}

	// Field (8) 'Eth1Data'
	if b.Eth1Data == nil {
		b.Eth1Data = new(Eth1Data)
	}
	if err = b.Eth1Data.DecodeSSZ(buf[524468:524540]); err != nil {
		return err
	}

	// Offset (9) 'Eth1DataVotes'
	if o9 = ssz.ReadOffset(buf[524540:524544]); o9 > size || o7 > o9 {
		return ssz.ErrOffset
	}

	// Field (10) 'Eth1DepositIndex'
	b.Eth1DepositIndex = ssz.UnmarshallUint64(buf[524544:524552])

	// Offset (11) 'Validators'
	if o11 = ssz.ReadOffset(buf[524552:524556]); o11 > size || o9 > o11 {
		return ssz.ErrOffset
	}

	// Offset (12) 'Balances'
	if o12 = ssz.ReadOffset(buf[524556:524560]); o12 > size || o11 > o12 {
		return ssz.ErrOffset
	}

	// Field (13) 'RandaoMixes'
	b.RandaoMixes = make([][32]byte, 65536)
	for ii := 0; ii < 65536; ii++ {
		copy(b.RandaoMixes[ii][:], buf[524560:2621712][ii*32:(ii+1)*32])
	}

	// Field (14) 'Slashings'
	b.Slashings = ssz.ExtendUint64(b.Slashings, 8192)
	for ii := 0; ii < 8192; ii++ {
		b.Slashings[ii] = ssz.UnmarshallUint64(buf[2621712:2687248][ii*8 : (ii+1)*8])
	}

	// Offset (15) 'PreviousEpochParticipation'
	if o15 = ssz.ReadOffset(buf[2687248:2687252]); o15 > size || o12 > o15 {
		return ssz.ErrOffset
	}

	// Offset (16) 'CurrentEpochParticipation'
	if o16 = ssz.ReadOffset(buf[2687252:2687256]); o16 > size || o15 > o16 {
		return ssz.ErrOffset
	}

	// Field (17) 'JustificationBits'
	if cap(b.JustificationBits) == 0 {
		b.JustificationBits = make([]byte, 0, len(buf[2687256:2687257]))
	}
	b.JustificationBits = append(b.JustificationBits, buf[2687256:2687257]...)

	// Field (18) 'PreviousJustifiedCheckpoint'
	if b.PreviousJustifiedCheckpoint == nil {
		b.PreviousJustifiedCheckpoint = new(Checkpoint)
	}
	if err = b.PreviousJustifiedCheckpoint.DecodeSSZ(buf[2687257:2687297]); err != nil {
		return err
	}

	// Field (19) 'CurrentJustifiedCheckpoint'
	if b.CurrentJustifiedCheckpoint == nil {
		b.CurrentJustifiedCheckpoint = new(Checkpoint)
	}
	if err = b.CurrentJustifiedCheckpoint.DecodeSSZ(buf[2687297:2687337]); err != nil {
		return err
	}

	// Field (20) 'FinalizedCheckpoint'
	if b.FinalizedCheckpoint == nil {
		b.FinalizedCheckpoint = new(Checkpoint)
	}
	if err = b.FinalizedCheckpoint.DecodeSSZ(buf[2687337:2687377]); err != nil {
		return err
	}

	// Offset (21) 'InactivityScores'
	if o21 = ssz.ReadOffset(buf[2687377:2687381]); o21 > size || o16 > o21 {
		return ssz.ErrOffset
	}

	// Field (22) 'CurrentSyncCommittee'
	if b.CurrentSyncCommittee == nil {
		b.CurrentSyncCommittee = new(SyncCommittee)
	}
	if err = b.CurrentSyncCommittee.UnmarshalSSZ(buf[2687381:2712005]); err != nil {
		return err
	}

	// Field (23) 'NextSyncCommittee'
	if b.NextSyncCommittee == nil {
		b.NextSyncCommittee = new(SyncCommittee)
	}
	if err = b.NextSyncCommittee.UnmarshalSSZ(buf[2712005:2736629]); err != nil {
		return err
	}

	// Offset (24) 'LatestExecutionPayloadHeader'
	if o24 = ssz.ReadOffset(buf[2736629:2736633]); o24 > size || o21 > o24 {
		return ssz.ErrOffset
	}

	// Field (7) 'HistoricalRoots'
	{
		buf = tail[o7:o9]
		num, err := ssz.DivideInt2(len(buf), 32, 16777216)
		if err != nil {
			return err
		}
		b.HistoricalRoots = make([][32]byte, num)
		for ii := 0; ii < num; ii++ {
			copy(b.HistoricalRoots[ii][:], buf[ii*32:(ii+1)*32])
		}
	}

	// Field (9) 'Eth1DataVotes'
	{
		buf = tail[o9:o11]
		num, err := ssz.DivideInt2(len(buf), 72, 2048)
		if err != nil {
			return err
		}
		b.Eth1DataVotes = make([]*Eth1Data, num)
		for ii := 0; ii < num; ii++ {
			if b.Eth1DataVotes[ii] == nil {
				b.Eth1DataVotes[ii] = new(Eth1Data)
			}
			if err = b.Eth1DataVotes[ii].DecodeSSZ(buf[ii*72 : (ii+1)*72]); err != nil {
				return err
			}
		}
	}

	// Field (11) 'Validators'
	{
		buf = tail[o11:o12]
		num, err := ssz.DivideInt2(len(buf), 121, 1099511627776)
		if err != nil {
			return err
		}
		b.Validators = make([]*Validator, num)
		for ii := 0; ii < num; ii++ {
			if b.Validators[ii] == nil {
				b.Validators[ii] = new(Validator)
			}
			if err = b.Validators[ii].UnmarshalSSZ(buf[ii*121 : (ii+1)*121]); err != nil {
				return err
			}
		}
	}

	// Field (12) 'Balances'
	{
		buf = tail[o12:o15]
		num, err := ssz.DivideInt2(len(buf), 8, 1099511627776)
		if err != nil {
			return err
		}
		b.Balances = ssz.ExtendUint64(b.Balances, num)
		for ii := 0; ii < num; ii++ {
			b.Balances[ii] = ssz.UnmarshallUint64(buf[ii*8 : (ii+1)*8])
		}
	}

	// Field (15) 'PreviousEpochParticipation'
	{
		buf = tail[o15:o16]
		if len(buf) > 1099511627776 {
			return ssz.ErrBytesLength
		}
		if cap(b.PreviousEpochParticipation) == 0 {
			b.PreviousEpochParticipation = make([]byte, 0, len(buf))
		}
		b.PreviousEpochParticipation = append(b.PreviousEpochParticipation, buf...)
	}

	// Field (16) 'CurrentEpochParticipation'
	{
		buf = tail[o16:o21]
		if len(buf) > 1099511627776 {
			return ssz.ErrBytesLength
		}
		if cap(b.CurrentEpochParticipation) == 0 {
			b.CurrentEpochParticipation = make([]byte, 0, len(buf))
		}
		b.CurrentEpochParticipation = append(b.CurrentEpochParticipation, buf...)
	}

	// Field (21) 'InactivityScores'
	{
		buf = tail[o21:o24]
		num, err := ssz.DivideInt2(len(buf), 8, 1099511627776)
		if err != nil {
			return err
		}
		b.InactivityScores = ssz.ExtendUint64(b.InactivityScores, num)
		for ii := 0; ii < num; ii++ {
			b.InactivityScores[ii] = ssz.UnmarshallUint64(buf[ii*8 : (ii+1)*8])
		}
	}

	// Field (24) 'LatestExecutionPayloadHeader'
	{
		buf = tail[o24:]
		if b.LatestExecutionPayloadHeader == nil {
			b.LatestExecutionPayloadHeader = new(types.Header)
		}
		if err = b.LatestExecutionPayloadHeader.DecodeSSZ(buf, clparams.BellatrixVersion); err != nil {
			return err
		}
	}
	return err
}

// SizeSSZ returns the ssz encoded size in bytes for the BeaconStateBellatrix object
func (b *BeaconStateBellatrix) SizeSSZ() (size int) {
	size = 2736633

	// Field (7) 'HistoricalRoots'
	size += len(b.HistoricalRoots) * 32

	// Field (9) 'Eth1DataVotes'
	size += len(b.Eth1DataVotes) * 72

	// Field (11) 'Validators'
	size += len(b.Validators) * 121

	// Field (12) 'Balances'
	size += len(b.Balances) * 8

	// Field (15) 'PreviousEpochParticipation'
	size += len(b.PreviousEpochParticipation)

	// Field (16) 'CurrentEpochParticipation'
	size += len(b.CurrentEpochParticipation)

	// Field (21) 'InactivityScores'
	size += len(b.InactivityScores) * 8

	// Field (24) 'LatestExecutionPayloadHeader'
	if b.LatestExecutionPayloadHeader == nil {
		b.LatestExecutionPayloadHeader = new(types.Header)
	}
	size += b.LatestExecutionPayloadHeader.EncodingSizeSSZ(clparams.BellatrixVersion)

	return
}

// HashTreeRoot ssz hashes the BeaconStateBellatrix object
func (b *BeaconStateBellatrix) HashTreeRoot() ([32]byte, error) {
	panic("Cannot use!")
}
