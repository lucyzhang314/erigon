package cltypes

import (
	"fmt"
	"math/big"

	libcommon "github.com/ledgerwatch/erigon-lib/common"
	"github.com/ledgerwatch/erigon-lib/types/ssz"

	"github.com/ledgerwatch/erigon/cl/clparams"
	"github.com/ledgerwatch/erigon/cl/cltypes/solid"
	"github.com/ledgerwatch/erigon/cl/merkle_tree"
	"github.com/ledgerwatch/erigon/consensus/merge"
	"github.com/ledgerwatch/erigon/core/types"
)

// ETH1Block represents a block structure CL-side.
type Eth1Block struct {
	ParentHash    libcommon.Hash
	FeeRecipient  libcommon.Address
	StateRoot     libcommon.Hash
	ReceiptsRoot  libcommon.Hash
	LogsBloom     types.Bloom
	PrevRandao    libcommon.Hash
	BlockNumber   uint64
	GasLimit      uint64
	GasUsed       uint64
	Time          uint64
	Extra         *solid.ExtraData
	BaseFeePerGas [32]byte
	// Extra fields
	BlockHash     libcommon.Hash
	Transactions  *solid.TransactionsSSZ
	Withdrawals   *solid.ListSSZ[*types.Withdrawal]
	ExcessDataGas [32]byte
	// internals
	version clparams.StateVersion
}

// NewEth1Block creates a new Eth1Block.
func NewEth1Block(version clparams.StateVersion) *Eth1Block {
	return &Eth1Block{version: version}
}

// NewEth1BlockFromHeaderAndBody with given header/body.
func NewEth1BlockFromHeaderAndBody(header *types.Header, body *types.RawBody) *Eth1Block {
	baseFeeBytes := header.BaseFee.Bytes()
	for i, j := 0, len(baseFeeBytes)-1; i < j; i, j = i+1, j-1 {
		baseFeeBytes[i], baseFeeBytes[j] = baseFeeBytes[j], baseFeeBytes[i]
	}
	var baseFee32 [32]byte
	copy(baseFee32[:], baseFeeBytes)

	var excessDataGas32 [32]byte
	if header.ExcessDataGas != nil {
		excessDataGasBytes := header.ExcessDataGas.Bytes()
		for i, j := 0, len(excessDataGasBytes)-1; i < j; i, j = i+1, j-1 {
			excessDataGasBytes[i], excessDataGasBytes[j] = excessDataGasBytes[j], excessDataGasBytes[i]
		}
		copy(excessDataGas32[:], excessDataGasBytes)
	}
	extra := solid.NewExtraData()
	extra.SetBytes(header.Extra)
	block := &Eth1Block{
		ParentHash:    header.ParentHash,
		FeeRecipient:  header.Coinbase,
		StateRoot:     header.Root,
		ReceiptsRoot:  header.ReceiptHash,
		LogsBloom:     header.Bloom,
		PrevRandao:    header.MixDigest,
		BlockNumber:   header.Number.Uint64(),
		GasLimit:      header.GasLimit,
		GasUsed:       header.GasUsed,
		Time:          header.Time,
		Extra:         extra,
		BaseFeePerGas: baseFee32,
		BlockHash:     header.Hash(),
		Transactions:  solid.NewTransactionsSSZFromTransactions(body.Transactions),
		Withdrawals:   solid.NewStaticListSSZFromList(body.Withdrawals, 16, 44),
		ExcessDataGas: excessDataGas32,
	}

	if header.ExcessDataGas != nil {
		block.version = clparams.DenebVersion
	} else if header.WithdrawalsHash != nil {
		block.version = clparams.CapellaVersion
	} else {
		block.version = clparams.BellatrixVersion
	}
	return block
}

// PayloadHeader returns the equivalent ExecutionPayloadHeader object.
func (b *Eth1Block) PayloadHeader() (*Eth1Header, error) {
	var err error
	var transactionsRoot, withdrawalsRoot libcommon.Hash
	if transactionsRoot, err = b.Transactions.HashSSZ(); err != nil {
		return nil, err
	}
	if b.version >= clparams.CapellaVersion {
		withdrawalsRoot, err = b.Withdrawals.HashSSZ()
		if err != nil {
			return nil, err
		}
	}

	return &Eth1Header{
		ParentHash:       b.ParentHash,
		FeeRecipient:     b.FeeRecipient,
		StateRoot:        b.StateRoot,
		ReceiptsRoot:     b.ReceiptsRoot,
		LogsBloom:        b.LogsBloom,
		PrevRandao:       b.PrevRandao,
		BlockNumber:      b.BlockNumber,
		GasLimit:         b.GasLimit,
		GasUsed:          b.GasUsed,
		Time:             b.Time,
		Extra:            b.Extra,
		BaseFeePerGas:    b.BaseFeePerGas,
		BlockHash:        b.BlockHash,
		TransactionsRoot: transactionsRoot,
		WithdrawalsRoot:  withdrawalsRoot,
		ExcessDataGas:    b.ExcessDataGas,
		version:          b.version,
	}, nil
}

// Return minimum required buffer length to be an acceptable SSZ encoding.
func (b *Eth1Block) EncodingSizeSSZ() (size int) {
	size = 508
	if b.Extra == nil {
		b.Extra = solid.NewExtraData()
	}
	// Field (10) 'ExtraData'
	size += b.Extra.EncodingSize()
	// Field (13) 'Transactions'
	size += b.Transactions.EncodingSize()

	if b.version >= clparams.CapellaVersion {
		if b.Withdrawals == nil {
			b.Withdrawals = solid.NewStaticListSSZ[*types.Withdrawal](16, 44)
		}
		size += b.Withdrawals.EncodingSizeSSZ() + 4
	}

	if b.version >= clparams.DenebVersion {
		size += 32 // ExcessDataGas
	}

	return
}

// DecodeSSZ decodes the block in SSZ format.
func (b *Eth1Block) DecodeSSZ(buf []byte, version int) error {
	b.version = clparams.StateVersion(version)
	if len(buf) < b.EncodingSizeSSZ() {
		return fmt.Errorf("[Eth1Block] err: %s", ssz.ErrLowBufferSize)
	}
	// We can reuse code from eth1-header for partial decoding
	payloadHeader := Eth1Header{}
	pos, extraDataOffset := payloadHeader.decodeHeaderMetadataForSSZ(buf)
	// Set all header shared fields accordingly
	b.ParentHash = payloadHeader.ParentHash
	b.FeeRecipient = payloadHeader.FeeRecipient
	b.StateRoot = payloadHeader.StateRoot
	b.ReceiptsRoot = payloadHeader.ReceiptsRoot
	b.BlockHash = payloadHeader.BlockHash
	b.LogsBloom = payloadHeader.LogsBloom
	b.PrevRandao = payloadHeader.PrevRandao
	b.BlockNumber = payloadHeader.BlockNumber
	b.GasLimit = payloadHeader.GasLimit
	b.GasUsed = payloadHeader.GasUsed
	b.Time = payloadHeader.Time
	b.BaseFeePerGas = payloadHeader.BaseFeePerGas
	// Decode the rest
	transactionsOffset := ssz.DecodeOffset(buf[pos:])
	pos += 4
	var withdrawalOffset *uint32
	if version >= int(clparams.CapellaVersion) {
		withdrawalOffset = new(uint32)
		*withdrawalOffset = ssz.DecodeOffset(buf[pos:])
	}
	pos += 4
	if version >= int(clparams.DenebVersion) {
		copy(b.ExcessDataGas[:], buf[pos:])
	}
	if b.Extra == nil {
		b.Extra = solid.NewExtraData()
	}
	// Compute extra data.
	if err := b.Extra.DecodeSSZ(buf[extraDataOffset:transactionsOffset], version); err != nil {
		return err
	}
	endOffset := uint32(len(buf))
	if withdrawalOffset != nil {
		endOffset = *withdrawalOffset
	}

	b.Transactions = new(solid.TransactionsSSZ)
	if err := b.Transactions.DecodeSSZ(buf[transactionsOffset:endOffset], version); err != nil {
		return err
	}

	// If withdrawals are enabled, process them.
	if withdrawalOffset != nil {
		if b.Withdrawals == nil {
			b.Withdrawals = solid.NewStaticListSSZ[*types.Withdrawal](16, 44)
		}
		if err := b.Withdrawals.DecodeSSZ(buf[*withdrawalOffset:], version); err != nil {
			return fmt.Errorf("[Eth1Block] err: %s", err)

		}
	}

	return nil
}

// EncodeSSZ encodes the block in SSZ format.
func (b *Eth1Block) EncodeSSZ(dst []byte) ([]byte, error) {
	buf := dst
	var err error
	currentOffset := ssz.BaseExtraDataSSZOffsetBlock

	if b.version >= clparams.CapellaVersion {
		currentOffset += 4
	}
	if b.version >= clparams.DenebVersion {
		currentOffset += 32
	}
	payloadHeader, err := b.PayloadHeader()
	if err != nil {
		return nil, err
	}
	buf, err = payloadHeader.encodeHeaderMetadataForSSZ(buf, currentOffset)
	if err != nil {
		return nil, err
	}
	if b.Extra == nil {
		b.Extra = solid.NewExtraData()
	}
	currentOffset += b.Extra.EncodingSize()
	// Write transaction offset
	buf = append(buf, ssz.OffsetSSZ(uint32(currentOffset))...)

	currentOffset += b.Transactions.EncodingSize()
	// Write withdrawals offset if exist
	if b.version >= clparams.CapellaVersion {
		buf = append(buf, ssz.OffsetSSZ(uint32(currentOffset))...)
		currentOffset += b.Withdrawals.EncodingSizeSSZ()
	}

	if b.version >= clparams.DenebVersion {
		buf = append(buf, b.ExcessDataGas[:]...)
	}

	buf = append(buf, b.Extra.Bytes()...)
	// Write all tx offsets
	if buf, err = b.Transactions.EncodeSSZ(buf); err != nil {
		return nil, err
	}
	if b.version < clparams.CapellaVersion {
		return buf, nil
	}

	return b.Withdrawals.EncodeSSZ(buf)
}

// HashSSZ calculates the SSZ hash of the Eth1Block's payload header.
func (b *Eth1Block) HashSSZ() ([32]byte, error) {
	switch b.version {
	case clparams.BellatrixVersion:
		return merkle_tree.HashTreeRoot(b.ParentHash[:], b.FeeRecipient[:], b.StateRoot[:], b.ReceiptsRoot[:], b.LogsBloom[:],
			b.PrevRandao[:], b.BlockNumber, b.GasLimit, b.GasUsed, b.Time, b.Extra, b.BaseFeePerGas[:], b.BlockHash[:], b.Transactions)
	case clparams.CapellaVersion:
		return merkle_tree.HashTreeRoot(b.ParentHash[:], b.FeeRecipient[:], b.StateRoot[:], b.ReceiptsRoot[:], b.LogsBloom[:],
			b.PrevRandao[:], b.BlockNumber, b.GasLimit, b.GasUsed, b.Time, b.Extra, b.BaseFeePerGas[:], b.BlockHash[:], b.Transactions,
			b.Withdrawals,
		)
	case clparams.DenebVersion:
		return merkle_tree.HashTreeRoot(b.ParentHash[:], b.FeeRecipient[:], b.StateRoot[:], b.ReceiptsRoot[:], b.LogsBloom[:],
			b.PrevRandao[:], b.BlockNumber, b.GasLimit, b.GasUsed, b.Time, b.Extra, b.BaseFeePerGas[:], b.BlockHash[:], b.Transactions,
			b.Withdrawals, b.ExcessDataGas[:],
		)
	default:
		panic("what do you want")
	}
}

// RlpHeader returns the equivalent types.Header struct with RLP-based fields.
func (b *Eth1Block) RlpHeader() (*types.Header, error) {
	// Reverse the order of the bytes in the BaseFeePerGas array and convert it to a big integer.
	reversedBaseFeePerGas := libcommon.Copy(b.BaseFeePerGas[:])
	for i, j := 0, len(reversedBaseFeePerGas)-1; i < j; i, j = i+1, j-1 {
		reversedBaseFeePerGas[i], reversedBaseFeePerGas[j] = reversedBaseFeePerGas[j], reversedBaseFeePerGas[i]
	}
	baseFee := new(big.Int).SetBytes(reversedBaseFeePerGas)

	// If the block version is Capella or later, calculate the withdrawals hash.
	var withdrawalsHash *libcommon.Hash
	if b.version >= clparams.CapellaVersion {
		withdrawalsHash = new(libcommon.Hash)
		// extract all withdrawals from itearable list
		withdrawals := make([]*types.Withdrawal, b.Withdrawals.Len())
		b.Withdrawals.Range(func(_ int, w *types.Withdrawal, _ int) bool {
			withdrawals = append(withdrawals, w)
			return true
		})
		*withdrawalsHash = types.DeriveSha(types.Withdrawals(withdrawals))
	}

	var excessDataGas *big.Int
	if b.version >= clparams.DenebVersion {
		reversedExcessDataGas := libcommon.Copy(b.ExcessDataGas[:])
		for i, j := 0, len(reversedExcessDataGas)-1; i < j; i, j = i+1, j-1 {
			reversedExcessDataGas[i], reversedExcessDataGas[j] = reversedExcessDataGas[j], reversedExcessDataGas[i]
		}
		excessDataGas = new(big.Int).SetBytes(reversedExcessDataGas)
	}

	header := &types.Header{
		ParentHash:      b.ParentHash,
		UncleHash:       types.EmptyUncleHash,
		Coinbase:        b.FeeRecipient,
		Root:            b.StateRoot,
		TxHash:          types.DeriveSha(types.BinaryTransactions(b.Transactions.UnderlyngReference())),
		ReceiptHash:     b.ReceiptsRoot,
		Bloom:           b.LogsBloom,
		Difficulty:      merge.ProofOfStakeDifficulty,
		Number:          big.NewInt(int64(b.BlockNumber)),
		GasLimit:        b.GasLimit,
		GasUsed:         b.GasUsed,
		Time:            b.Time,
		Extra:           b.Extra.Bytes(),
		MixDigest:       b.PrevRandao,
		Nonce:           merge.ProofOfStakeNonce,
		BaseFee:         baseFee,
		WithdrawalsHash: withdrawalsHash,
		ExcessDataGas:   excessDataGas,
	}

	// If the header hash does not match the block hash, return an error.
	if header.Hash() != b.BlockHash {
		return nil, fmt.Errorf("cannot derive rlp header: mismatching hash")
	}

	return header, nil
}

// Body returns the equivalent raw body (only eth1 body section).
func (b *Eth1Block) Body() *types.RawBody {
	withdrawals := make([]*types.Withdrawal, b.Withdrawals.Len())
	b.Withdrawals.Range(func(_ int, w *types.Withdrawal, _ int) bool {
		withdrawals = append(withdrawals, w)
		return true
	})
	return &types.RawBody{
		Transactions: b.Transactions.UnderlyngReference(),
		Withdrawals:  types.Withdrawals(withdrawals),
	}
}
