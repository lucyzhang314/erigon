package params

var (
	MainnetChainSnapshotConfig = &SnapshotsConfig{}
	GoerliChainSnapshotConfig  = &SnapshotsConfig{
		ExpectBlocks: 5_500_000,
	}
)

type SnapshotsConfig struct {
	ExpectBlocks uint64
}

func KnownSnapshots(networkName string) *SnapshotsConfig {
	switch networkName {
	case MainnetChainName:
		return MainnetChainSnapshotConfig
	case GoerliChainName:
		return GoerliChainSnapshotConfig
	default:
		return nil
	}
}
