package consensus_tests

import (
	"github.com/ledgerwatch/erigon/cl/clparams"
	"github.com/ledgerwatch/erigon/cl/cltypes"
	"github.com/ledgerwatch/erigon/cmd/ef-tests-cl/spectest"
	"github.com/ledgerwatch/erigon/cmd/erigon-cl/core/state"
)

var TestFormats = spectest.Appendix{}

func init() {
	TestFormats.Add("bls").
		With("aggregate_verify", &BlsAggregateVerify{}).
		With("aggregate", spectest.UnimplementedHandler).
		With("eth_aggregate_pubkeys", spectest.UnimplementedHandler).
		With("eth_fast_aggregate_verify", spectest.UnimplementedHandler).
		With("fast_aggregate_verify", spectest.UnimplementedHandler).
		With("sign", spectest.UnimplementedHandler).
		With("verify", spectest.UnimplementedHandler)
	TestFormats.Add("epoch_processing").
		With("effective_balance_updates", effectiveBalancesUpdateTest).
		With("eth1_data_reset", eth1DataResetTest).
		With("historical_roots_update", historicalRootsUpdateTest).
		With("inactivity_updates", inactivityUpdateTest).
		With("justification_and_finalization", justificationFinalizationTest).
		With("participation_flag_updates", participationFlagUpdatesTest).
		With("randao_mixes_reset", randaoMixesTest).
		With("registry_updates", registryUpdatesTest).
		With("rewards_and_penalties", rewardsAndPenaltiesTest).
		With("slashings", slashingsTest).
		With("slashings_reset", slashingsResetTest).
		With("participation_record_updates", participationFlagUpdatesTest)
	TestFormats.Add("finality").
		With("finality", FinalityFinality)
	TestFormats.Add("fork_choice").
		With("get_head", &ForkChoice{}).
		With("on_block", &ForkChoice{}).
		With("on_merge_block", &ForkChoice{}).
		With("ex_ante", &ForkChoice{})
	TestFormats.Add("forks").
		With("fork", ForksFork)
	TestFormats.Add("genesis").
		With("validity", spectest.UnimplementedHandler).
		With("initialization", spectest.UnimplementedHandler)
	TestFormats.Add("kzg").
		With("", spectest.UnimplementedHandler)
	TestFormats.Add("light_client").
		With("", spectest.UnimplementedHandler)
	TestFormats.Add("operations").
		WithFn("attestation", operationAttestationHandler).
		WithFn("attester_slashing", operationAttesterSlashingHandler).
		WithFn("proposer_slashing", operationProposerSlashingHandler).
		WithFn("block_header", operationBlockHeaderHandler).
		WithFn("deposit", operationDepositHandler).
		WithFn("voluntary_exit", operationVoluntaryExitHandler).
		WithFn("sync_aggregate", operationSyncAggregateHandler).
		WithFn("withdrawals", operationWithdrawalHandler).
		WithFn("bls_to_execution-change", operationSignedBlsChangeHandler)
	TestFormats.Add("random").
		With("", spectest.UnimplementedHandler)
	TestFormats.Add("rewards").
		With("", spectest.UnimplementedHandler)
	TestFormats.Add("sanity").
		With("slots", SanitySlots).
		With("blocks", SanityBlocks)
	TestFormats.Add("shuffling").
		With("", spectest.UnimplementedHandler)
	TestFormats.Add("ssz_generic").
		With("", spectest.UnimplementedHandler)
	TestFormats.Add("sync").
		With("", spectest.UnimplementedHandler)
	TestFormats.Add("transition").
		With("core", &TransitionCore{})

	addSszTests()
}

func addSszTests() {
	TestFormats.Add("ssz_static").
		//	With("AggregateAndProof", getSSZStaticConsensusTest(&cltypes.AggregateAndProof{})).
		With("Attestation", getSSZStaticConsensusTest(&cltypes.Attestation{})).
		//		With("AttestationData", getSSZStaticConsensusTest(&cltypes.AttestationData{})).
		With("AttesterSlashing", getSSZStaticConsensusTest(&cltypes.AttesterSlashing{})).
		With("BeaconBlock", getSSZStaticConsensusTest(&cltypes.BeaconBlock{})).
		//With("BeaconBlockBody", getSSZStaticConsensusTest(&cltypes.BeaconBlockBody{})).
		//		With("BeaconBlockHeader", getSSZStaticConsensusTest(&cltypes.BeaconBlockHeader{})).
		With("BeaconState", getSSZStaticConsensusTest(state.New(&clparams.MainnetBeaconConfig))).
		//		With("BlobIdentifier", getSSZStaticConsensusTest(&cltypes.BlobIdentifier{})).
		//		With("BlobSidecar", getSSZStaticConsensusTest(&cltypes.BlobSidecar{})).
		//		With("BLSToExecutionChange", getSSZStaticConsensusTest(&cltypes.BLSToExecutionChange{})).
		With("Checkpoint", getSSZStaticConsensusTest(&cltypes.Checkpoint{})).
		//	With("ContributionAndProof", getSSZStaticConsensusTest(&cltypes.ContributionAndProof{})).
		With("Deposit", getSSZStaticConsensusTest(&cltypes.Deposit{})).
		With("DepositData", getSSZStaticConsensusTest(&cltypes.DepositData{})).
		//	With("DepositMessage", getSSZStaticConsensusTest(&cltypes.DepositMessage{})).
		//	With("Eth1Block", getSSZStaticConsensusTest(&cltypes.Eth1Block{})).
		With("Eth1Data", getSSZStaticConsensusTest(&cltypes.Eth1Data{})).
		//With("ExecutionPayload", getSSZStaticConsensusTest(&cltypes.ExecutionPayload{})).
		//With("ExecutionPayloadHeader", getSSZStaticConsensusTest(&cltypes.ExecutionPayloadHeader{})).
		//With("Fork", getSSZStaticConsensusTest(&cltypes.Fork{})).
		//With("ForkData", getSSZStaticConsensusTest(&cltypes.ForkData{})).
		//With("HistoricalBatch", getSSZStaticConsensusTest(&cltypes.HistoricalBatch{})).
		With("HistoricalSummary", getSSZStaticConsensusTest(&cltypes.HistoricalSummary{})).
		//	With("IndexedAttestation", getSSZStaticConsensusTest(&cltypes.IndexedAttestation{})).
		//	With("LightClientBootstrap", getSSZStaticConsensusTest(&cltypes.LightClientBootstrap{})).
		//	With("LightClientFinalityUpdate", getSSZStaticConsensusTest(&cltypes.LightClientFinalityUpdate{})).
		//	With("LightClientHeader", getSSZStaticConsensusTest(&cltypes.LightClientHeader{})).
		//	With("LightClientOptimisticUpdate", getSSZStaticConsensusTest(&cltypes.LightClientOptimisticUpdate{})).
		//	With("LightClientUpdate", getSSZStaticConsensusTest(&cltypes.LightClientUpdate{})).
		With("PendingAttestation", getSSZStaticConsensusTest(&cltypes.PendingAttestation{})).
		//		With("PowBlock", getSSZStaticConsensusTest(&cltypes.PowBlock{})).
		With("ProposerSlashing", getSSZStaticConsensusTest(&cltypes.ProposerSlashing{})).
		//		With("SignedAggregateAndProof", getSSZStaticConsensusTest(&cltypes.SignedAggregateAndProof{})).
		With("SignedBeaconBlock", getSSZStaticConsensusTest(&cltypes.SignedBeaconBlock{})).
		//	With("SignedBeaconBlockHeader", getSSZStaticConsensusTest(&cltypes.SignedBeaconBlockHeader{})).
		//	With("SignedBlobSidecar", getSSZStaticConsensusTest(&cltypes.SignedBlobSidecar{})).
		With("SignedBLSToExecutionChange", getSSZStaticConsensusTest(&cltypes.SignedBLSToExecutionChange{})).
		//		With("SignedContributionAndProof", getSSZStaticConsensusTest(&cltypes.SignedContributionAndProof{})).
		//	With("SignedVoluntaryExit", getSSZStaticConsensusTest(&cltypes.SignedVoluntaryExit{})).
		//	With("SigningData", getSSZStaticConsensusTest(&cltypes.SigningData{})).
		//	With("SyncAggregate", getSSZStaticConsensusTest(&cltypes.SyncAggregate{})).
		//	With("SyncAggregatorSelectionData", getSSZStaticConsensusTest(&cltypes.SyncAggregatorSelectionData{})).
		//	With("SyncCommittee", getSSZStaticConsensusTest(&cltypes.SyncCommittee{})).
		//	With("SyncCommitteeContribution", getSSZStaticConsensusTest(&cltypes.SyncCommitteeContribution{})).
		//	With("SyncCommitteeMessage", getSSZStaticConsensusTest(&cltypes.SyncCommitteeMessage{})).
		With("Validator", getSSZStaticConsensusTest(&cltypes.Validator{}))
	//	With("VoluntaryExit", getSSZStaticConsensusTest(&cltypes.VoluntaryExit{})).
	//	With("Withdrawal", getSSZStaticConsensusTest(&types.Withdrawal{}))
}
