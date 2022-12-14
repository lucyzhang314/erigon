package span

import (
	"encoding/hex"
	"math/big"

	"github.com/ledgerwatch/erigon/accounts/abi"
	"github.com/ledgerwatch/erigon/common"
	"github.com/ledgerwatch/erigon/consensus"
	"github.com/ledgerwatch/erigon/consensus/bor/valset"
	"github.com/ledgerwatch/erigon/params"
	"github.com/ledgerwatch/erigon/params/networkname"
	"github.com/ledgerwatch/erigon/rlp"
	"github.com/ledgerwatch/log/v3"
)

type ChainSpanner struct {
	validatorSet             abi.ABI
	chainConfig              *params.ChainConfig
	validatorContractAddress common.Address
}

func NewChainSpanner(validatorSet abi.ABI, chainConfig *params.ChainConfig, validatorContractAddress common.Address) *ChainSpanner {
	return &ChainSpanner{
		validatorSet:             validatorSet,
		chainConfig:              chainConfig,
		validatorContractAddress: validatorContractAddress,
	}
}

// GetCurrentSpan get current span from contract
func (c *ChainSpanner) GetCurrentSpan(syscall consensus.SystemCall) (*Span, error) {

	// method
	const method = "getCurrentSpan"

	data, err := c.validatorSet.Pack(method)
	if err != nil {
		log.Error("Unable to pack tx for getCurrentSpan", "err", err)
		return nil, err
	}

	result, err := syscall(common.HexToAddress(c.chainConfig.Bor.ValidatorContract), data)
	if err != nil {
		return nil, err
	}

	// span result
	ret := new(struct {
		Number     *big.Int
		StartBlock *big.Int
		EndBlock   *big.Int
	})

	if err := c.validatorSet.UnpackIntoInterface(ret, method, result); err != nil {
		return nil, err
	}

	// create new span
	span := Span{
		ID:         ret.Number.Uint64(),
		StartBlock: ret.StartBlock.Uint64(),
		EndBlock:   ret.EndBlock.Uint64(),
	}

	return &span, nil
}

func (c *ChainSpanner) GetCurrentValidators(blockNumber uint64, signer common.Address, getSpanForBlock func(blockNum uint64) (*HeimdallSpan, error)) ([]*valset.Validator, error) {
	// Use signer as validator in case of bor devent
	if c.chainConfig.ChainName == networkname.BorDevnetChainName {
		validators := []*valset.Validator{
			{
				ID:               1,
				Address:          signer,
				VotingPower:      1000,
				ProposerPriority: 1,
			},
		}

		return validators, nil
	}

	span, err := getSpanForBlock(blockNumber)
	if err != nil {
		return nil, err
	}

	return span.ValidatorSet.Validators, nil
}

const method = "commitSpan"

func (c *ChainSpanner) CommitSpan(heimdallSpan HeimdallSpan, syscall consensus.SystemCall) error {
	// get validators bytes
	validators := make([]valset.MinimalVal, 0, len(heimdallSpan.ValidatorSet.Validators))
	for _, val := range heimdallSpan.ValidatorSet.Validators {
		validators = append(validators, val.MinimalVal())
	}
	validatorBytes, err := rlp.EncodeToBytes(validators)
	if err != nil {
		return err
	}

	// get producers bytes
	producers := make([]valset.MinimalVal, 0, len(heimdallSpan.SelectedProducers))
	for _, val := range heimdallSpan.SelectedProducers {
		producers = append(producers, val.MinimalVal())
	}
	producerBytes, err := rlp.EncodeToBytes(producers)
	if err != nil {
		return err
	}

	log.Debug("✅ Committing new span",
		"id", heimdallSpan.ID,
		"startBlock", heimdallSpan.StartBlock,
		"endBlock", heimdallSpan.EndBlock,
		"validatorBytes", hex.EncodeToString(validatorBytes),
		"producerBytes", hex.EncodeToString(producerBytes),
	)

	// get packed data
	data, err := c.validatorSet.Pack(method,
		big.NewInt(0).SetUint64(heimdallSpan.ID),
		big.NewInt(0).SetUint64(heimdallSpan.StartBlock),
		big.NewInt(0).SetUint64(heimdallSpan.EndBlock),
		validatorBytes,
		producerBytes,
	)
	if err != nil {
		log.Error("Unable to pack tx for commitSpan", "err", err)
		return err
	}

	_, err = syscall(common.HexToAddress(c.chainConfig.Bor.ValidatorContract), data)
	// apply message
	return err
}
