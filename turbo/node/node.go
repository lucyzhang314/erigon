// Package node contains classes for running a Erigon node.
package node

import (
	"github.com/ledgerwatch/erigon-lib/kv"
	"github.com/ledgerwatch/erigon/cmd/utils"
	"github.com/ledgerwatch/erigon/eth"
	"github.com/ledgerwatch/erigon/eth/ethconfig"
	"github.com/ledgerwatch/erigon/node"
	"github.com/ledgerwatch/erigon/params"
	erigoncli "github.com/ledgerwatch/erigon/turbo/cli"
	"github.com/ledgerwatch/log/v3"

	"github.com/urfave/cli"
)

// ErigonNode represents a single node, that runs sync and p2p network.
// it also can export the private endpoint for RPC daemon, etc.
type ErigonNode struct {
	stack   *node.Node
	backend *eth.Ethereum
}

// Serve runs the node and blocks the execution. It returns when the node is existed.
func (eri *ErigonNode) Serve() error {
	defer eri.stack.Close()

	eri.run()

	eri.stack.Wait()

	return nil
}

func (eri *ErigonNode) run() {
	utils.StartNode(eri.stack)
	// we don't have accounts locally and we don't do mining
	// so these parts are ignored
	// see cmd/geth/main.go#startNode for full implementation
}

// Params contains optional parameters for creating a node.
// * GitCommit is a commit from which then node was built.
// * CustomBuckets is a `map[string]dbutils.TableCfgItem`, that contains bucket name and its properties.
//
// NB: You have to declare your custom buckets here to be able to use them in the app.
type Params struct {
	CustomBuckets kv.TableCfg
}

func NewNodConfigUrfave(ctx *cli.Context) *node.Config {
	// If we're running a known preset, log it for convenience.
	chain := ctx.GlobalString(utils.ChainFlag.Name)
	switch chain {
	case params.RopstenChainName:
		log.Info("Starting TurboBor on Ropsten testnet...")

	case params.RinkebyChainName:
		log.Info("Starting TurboBor on Rinkeby testnet...")

	case params.GoerliChainName:
		log.Info("Starting TurboBor on Görli testnet...")

	case params.DevChainName:
		log.Info("Starting TurboBor in ephemeral dev mode...")

	case params.MumbaiChainName:
		log.Info("Starting TurboBor in Mumbai testnet")

	case params.BorMainnetChainName:
		log.Info("Starting TurboBor on Bor Mainnet")

	case "", params.MainnetChainName:
		if !ctx.GlobalIsSet(utils.NetworkIdFlag.Name) {
			log.Info("Starting TurboBor on Ethereum mainnet...")
		}
	default:
		log.Info("Starting TurboBor on", "devnet", chain)
	}

	nodeConfig := NewNodeConfig()
	utils.SetNodeConfig(ctx, nodeConfig)
	erigoncli.ApplyFlagsForNodeConfig(ctx, nodeConfig)
	return nodeConfig
}
func NewEthConfigUrfave(ctx *cli.Context, nodeConfig *node.Config) *ethconfig.Config {
	ethConfig := &ethconfig.Defaults
	utils.SetEthConfig(ctx, nodeConfig, ethConfig)
	erigoncli.ApplyFlagsForEthConfig(ctx, ethConfig)
	return ethConfig
}

// New creates a new `ErigonNode`.
// * ctx - `*cli.Context` from the main function. Necessary to be able to configure the node based on the command-line flags
// * sync - `stagedsync.StagedSync`, an instance of staged sync, setup just as needed.
// * optionalParams - additional parameters for running a node.
func New(
	nodeConfig *node.Config,
	ethConfig *ethconfig.Config,
	logger log.Logger,
) (*ErigonNode, error) {
	//prepareBuckets(optionalParams.CustomBuckets)
	node := makeConfigNode(nodeConfig)
	ethereum, err := RegisterEthService(node, ethConfig, logger)
	if err != nil {
		return nil, err
	}
	return &ErigonNode{stack: node, backend: ethereum}, nil
}

// RegisterEthService adds an Ethereum client to the stack.
func RegisterEthService(stack *node.Node, cfg *ethconfig.Config, logger log.Logger) (*eth.Ethereum, error) {
	return eth.New(stack, cfg, logger)
}

func NewNodeConfig() *node.Config {
	nodeConfig := node.DefaultConfig
	// see simiar changes in `cmd/geth/config.go#defaultNodeConfig`
	if commit := params.GitCommit; commit != "" {
		nodeConfig.Version = params.VersionWithCommit(commit, "")
	} else {
		nodeConfig.Version = params.Version
	}
	nodeConfig.IPCPath = "" // force-disable IPC endpoint
	nodeConfig.Name = "erigon"
	return &nodeConfig
}

func MakeConfigNodeDefault() *node.Node {
	return makeConfigNode(NewNodeConfig())
}

func makeConfigNode(config *node.Config) *node.Node {
	stack, err := node.New(config)
	if err != nil {
		utils.Fatalf("Failed to create TurboBor node: %v", err)
	}

	return stack
}
