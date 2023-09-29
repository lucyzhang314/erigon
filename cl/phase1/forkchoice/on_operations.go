package forkchoice

import (
	"errors"
	"fmt"

	"github.com/Giulio2002/bls"
	"github.com/ledgerwatch/erigon/cl/cltypes"
	"github.com/ledgerwatch/erigon/cl/fork"
	"github.com/ledgerwatch/erigon/cl/phase1/core/state"
	"github.com/ledgerwatch/erigon/cl/pool"
)

// NOTE: This file implements non-official handlers for other types of iterations. what it does is,using the forkchoices
// and verify external operations and eventually push them in the operations pool.

// OnVoluntaryExit is a non-official handler for voluntary exit operations. it pushes the voluntary exit in the pool.
func (f *ForkChoiceStore) OnVoluntaryExit(signedVoluntaryExit *cltypes.SignedVoluntaryExit, test bool) (invalid bool, err error) {
	voluntaryExit := signedVoluntaryExit.VoluntaryExit
	if f.operationsPool.VoluntaryExistsPool.Has(voluntaryExit.ValidatorIndex) {
		return false, nil
	}
	f.mu.Lock()

	headHash, _, err := f.getHead()
	if err != nil {
		f.mu.Unlock()
		return false, err
	}
	s, _, err := f.forkGraph.GetState(headHash, false)
	if err != nil {
		f.mu.Unlock()
		return false, err
	}

	val, err := s.ValidatorForValidatorIndex(int(voluntaryExit.ValidatorIndex))
	if err != nil {
		f.mu.Unlock()
		return true, err
	}

	if val.ExitEpoch() != f.forkGraph.Config().FarFutureEpoch {
		f.mu.Unlock()
		return false, nil
	}

	pk := val.PublicKey()
	f.mu.Unlock()

	domain, err := s.GetDomain(s.BeaconConfig().DomainVoluntaryExit, voluntaryExit.Epoch)
	if err != nil {
		return false, err
	}
	signingRoot, err := fork.ComputeSigningRoot(voluntaryExit, domain)
	if err != nil {
		return true, err
	}
	if !test {
		valid, err := bls.Verify(signedVoluntaryExit.Signature[:], signingRoot[:], pk[:])
		if err != nil {
			return true, err
		}
		if !valid {
			return true, errors.New("ProcessVoluntaryExit: BLS verification failed")
		}
	}
	f.operationsPool.VoluntaryExistsPool.Insert(voluntaryExit.ValidatorIndex, signedVoluntaryExit)
	return false, nil
}

// OnProposerSlashing is a non-official handler for proposer slashing operations. it pushes the proposer slashing in the pool.
func (f *ForkChoiceStore) OnProposerSlashing(proposerSlashing *cltypes.ProposerSlashing, test bool) (err error) {
	if f.operationsPool.ProposerSlashingsPool.Has(pool.ComputeKeyForProposerSlashing(proposerSlashing)) {
		return nil
	}
	h1 := proposerSlashing.Header1.Header
	h2 := proposerSlashing.Header2.Header

	if h1.Slot != h2.Slot {
		return fmt.Errorf("non-matching slots on proposer slashing: %d != %d", h1.Slot, h2.Slot)
	}

	if h1.ProposerIndex != h2.ProposerIndex {
		return fmt.Errorf("non-matching proposer indices proposer slashing: %d != %d", h1.ProposerIndex, h2.ProposerIndex)
	}

	if *h1 == *h2 {
		return fmt.Errorf("proposee slashing headers are the same")
	}

	// Take lock as we interact with state.
	f.mu.Lock()
	headHash, _, err := f.getHead()
	if err != nil {
		f.mu.Unlock()
		return err
	}
	s, _, err := f.forkGraph.GetState(headHash, false)
	if err != nil {
		f.mu.Unlock()
		return err
	}
	proposer, err := s.ValidatorForValidatorIndex(int(h1.ProposerIndex))
	if err != nil {
		f.mu.Unlock()
		return fmt.Errorf("unable to retrieve state: %v", err)
	}
	if !proposer.IsSlashable(state.Epoch(s)) {
		f.mu.Unlock()
		return fmt.Errorf("proposer is not slashable: %v", proposer)
	}
	domain1, err := s.GetDomain(s.BeaconConfig().DomainBeaconProposer, state.GetEpochAtSlot(s.BeaconConfig(), h1.Slot))
	if err != nil {
		return fmt.Errorf("unable to get domain: %v", err)
	}
	domain2, err := s.GetDomain(s.BeaconConfig().DomainBeaconProposer, state.GetEpochAtSlot(s.BeaconConfig(), h2.Slot))
	if err != nil {
		return fmt.Errorf("unable to get domain: %v", err)
	}
	pk := proposer.PublicKey()
	f.mu.Unlock()
	if test {
		f.operationsPool.ProposerSlashingsPool.Insert(pool.ComputeKeyForProposerSlashing(proposerSlashing), proposerSlashing)
		return nil
	}
	signingRoot, err := fork.ComputeSigningRoot(h1, domain1)
	if err != nil {
		return fmt.Errorf("unable to compute signing root: %v", err)
	}
	valid, err := bls.Verify(proposerSlashing.Header1.Signature[:], signingRoot[:], pk[:])
	if err != nil {
		return fmt.Errorf("unable to verify signature: %v", err)
	}
	if !valid {
		return fmt.Errorf("invalid signature: signature %v, root %v, pubkey %v", proposerSlashing.Header1.Signature[:], signingRoot[:], pk)
	}
	signingRoot, err = fork.ComputeSigningRoot(h2, domain2)
	if err != nil {
		return fmt.Errorf("unable to compute signing root: %v", err)
	}

	valid, err = bls.Verify(proposerSlashing.Header2.Signature[:], signingRoot[:], pk[:])
	if err != nil {
		return fmt.Errorf("unable to verify signature: %v", err)
	}
	if !valid {
		return fmt.Errorf("invalid signature: signature %v, root %v, pubkey %v", proposerSlashing.Header2.Signature[:], signingRoot[:], pk)
	}
	f.operationsPool.ProposerSlashingsPool.Insert(pool.ComputeKeyForProposerSlashing(proposerSlashing), proposerSlashing)

	return nil
}
