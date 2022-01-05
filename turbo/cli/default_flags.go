package cli

import (
	"github.com/ledgerwatch/erigon/cmd/utils"

	"github.com/urfave/cli"
)

// DefaultFlags contains all flags that are used and supported by Erigon binary.
var DefaultFlags = []cli.Flag{
	utils.DataDirFlag,
	utils.MdbxAugmentLimitFlag,
	utils.EthashDatasetDirFlag,
	utils.TxPoolDisableFlag,
	utils.TxPoolLocalsFlag,
	utils.TxPoolNoLocalsFlag,
	utils.TxPoolJournalFlag,
	utils.TxPoolRejournalFlag,
	utils.TxPoolPriceLimitFlag,
	utils.TxPoolPriceBumpFlag,
	utils.TxPoolAccountSlotsFlag,
	utils.TxPoolGlobalSlotsFlag,
	utils.TxPoolGlobalBaseFeeSlotsFlag,
	utils.TxPoolAccountQueueFlag,
	utils.TxPoolGlobalQueueFlag,
	utils.TxPoolLifetimeFlag,
	utils.TxPoolTraceSendersFlag,
	PruneFlag,
	PruneHistoryFlag,
	PruneReceiptFlag,
	PruneTxIndexFlag,
	PruneCallTracesFlag,
	PruneHistoryBeforeFlag,
	PruneReceiptBeforeFlag,
	PruneTxIndexBeforeFlag,
	PruneCallTracesBeforeFlag,
	BatchSizeFlag,
	BlockDownloaderWindowFlag,
	DatabaseVerbosityFlag,
	PrivateApiAddr,
	PrivateApiRateLimit,
	EtlBufferSizeFlag,
	TLSFlag,
	TLSCertFlag,
	TLSKeyFlag,
	TLSCACertFlag,
	StateStreamDisableFlag,
	SyncLoopThrottleFlag,
	BadBlockFlag,
	utils.SnapshotSyncFlag,
	utils.ListenPortFlag,
	utils.NATFlag,
	utils.NoDiscoverFlag,
	utils.DiscoveryV5Flag,
	utils.NetrestrictFlag,
	utils.NodeKeyFileFlag,
	utils.NodeKeyHexFlag,
	utils.DNSDiscoveryFlag,
	utils.StaticPeersFlag,
	utils.TrustedPeersFlag,
	utils.MaxPeersFlag,
	utils.ChainFlag,
	utils.DeveloperPeriodFlag,
	utils.VMEnableDebugFlag,
	utils.NetworkIdFlag,
	utils.FakePoWFlag,
	utils.GpoBlocksFlag,
	utils.GpoPercentileFlag,
	utils.InsecureUnlockAllowedFlag,
	utils.MetricsEnabledFlag,
	utils.MetricsEnabledExpensiveFlag,
	utils.MetricsHTTPFlag,
	utils.MetricsPortFlag,
	utils.IdentityFlag,
	utils.CliqueSnapshotCheckpointIntervalFlag,
	utils.CliqueSnapshotInmemorySnapshotsFlag,
	utils.CliqueSnapshotInmemorySignaturesFlag,
	utils.CliqueDataDirFlag,
	utils.MiningEnabledFlag,
	utils.ProposingEnabledFlag,
	utils.MinerNotifyFlag,
	utils.MinerGasLimitFlag,
	utils.MinerEtherbaseFlag,
	utils.MinerExtraDataFlag,
	utils.MinerNoVerfiyFlag,
	utils.MinerSigningKeyFileFlag,
	utils.SentryAddrFlag,
	utils.DownloaderAddrFlag,
	HealthCheckFlag,
}
