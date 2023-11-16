// Copyright 2015 The go-ethereum Authors
// This file is part of the go-ethereum library.
//
// The go-ethereum library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-ethereum library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-ethereum library. If not, see <http://www.gnu.org/licenses/>.

package rpc

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"math/big"
	"strconv"
	"strings"

	"github.com/ledgerwatch/erigon-lib/common/hexutil"

	libcommon "github.com/ledgerwatch/erigon-lib/common"
)

// API describes the set of methods offered over the RPC interface
type API struct {
	Namespace string      // namespace under which the rpc methods of Service are exposed
	Version   string      // api version for DApp's
	Service   interface{} // receiver instance which holds the methods
	Public    bool        // indication if the methods must be considered safe for public use
}

// Error wraps RPC errors, which contain an error code in addition to the message.
type Error interface {
	Error() string  // returns the message
	ErrorCode() int // returns the code
}

// A DataError contains some data in addition to the error message.
type DataError interface {
	Error() string          // returns the message
	ErrorData() interface{} // returns the error data
}

// ServerCodec implements reading, parsing and writing RPC messages for the server side of
// a RPC session. Implementations must be go-routine safe since the codec can be called in
// multiple go-routines concurrently.
type ServerCodec interface {
	ReadBatch() (msgs []*jsonrpcMessage, isBatch bool, err error)
	Close()
	jsonWriter
}

// jsonWriter can write JSON messages to its underlying connection.
// Implementations must be safe for concurrent use.
type jsonWriter interface {
	writeJSON(context.Context, interface{}) error
	// Closed returns a channel which is closed when the connection is closed.
	closed() <-chan interface{}
	// RemoteAddr returns the peer address of the connection.
	remoteAddr() string
}

type BlockNumber int64
type Timestamp uint64

const (
	LatestExecutedBlockNumber = BlockNumber(-5)
	FinalizedBlockNumber      = BlockNumber(-4)
	SafeBlockNumber           = BlockNumber(-3)
	PendingBlockNumber        = BlockNumber(-2)
	LatestBlockNumber         = BlockNumber(-1)
	EarliestBlockNumber       = BlockNumber(0)
)

var (
	LatestExecutedBlock = LatestExecutedBlockNumber.AsBlockReference()
	FinalizedBlock      = FinalizedBlockNumber.AsBlockReference()
	SafeBlock           = SafeBlockNumber.AsBlockReference()
	PendingBlock        = PendingBlockNumber.AsBlockReference()
	LatestBlock         = LatestBlockNumber.AsBlockReference()
	EarliestBlock       = EarliestBlockNumber.AsBlockReference()
)

// UnmarshalJSON parses the given JSON fragment into a BlockNumber. It supports:
// - "latest", "earliest", "pending", "safe", or "finalized" as string arguments
// - the block number
// Returned errors:
// - an invalid block number error when the given argument isn't a known strings
// - an out of range error when the given block number is either too little or too large
func (bn *BlockNumber) UnmarshalJSON(data []byte) error {
	input := strings.TrimSpace(string(data))
	if len(input) >= 2 && input[0] == '"' && input[len(input)-1] == '"' {
		input = input[1 : len(input)-1]
	}

	switch input {
	case "earliest":
		*bn = EarliestBlockNumber
		return nil
	case "latest":
		*bn = LatestBlockNumber
		return nil
	case "pending":
		*bn = PendingBlockNumber
		return nil
	case "safe":
		*bn = SafeBlockNumber
		return nil
	case "finalized":
		*bn = FinalizedBlockNumber
		return nil
	case "latestExecuted":
		*bn = LatestExecutedBlockNumber
		return nil
	case "null":
		*bn = LatestBlockNumber
		return nil
	}

	// Try to parse it as a number
	blckNum, err := strconv.ParseUint(input, 10, 64)
	if err != nil {
		// Now try as a hex number
		if blckNum, err = hexutil.DecodeUint64(input); err != nil {
			return err
		}
	}
	if blckNum > math.MaxInt64 {
		return fmt.Errorf("block number larger than int64")
	}
	*bn = BlockNumber(blckNum)
	return nil
}

func (bn BlockNumber) MarshalText() ([]byte, error) {
	switch {
	case bn < LatestExecutedBlockNumber:
		return nil, fmt.Errorf("Invalid block number %d", bn)
	case bn < 0:
		return []byte(bn.String()), nil
	default:
		return []byte(bn.string(16)), nil
	}
}

func (bn BlockNumber) Int64() int64 {
	return int64(bn)
}

func (bn BlockNumber) Uint64() uint64 {
	return uint64(bn)
}

func (bn BlockNumber) String() string {
	return bn.string(10)
}

func (bn BlockNumber) AsBlockReference() BlockReference {
	return AsBlockReference(bn)
}

func (bn BlockNumber) string(base int) string {
	switch bn {
	case EarliestBlockNumber:
		return "earliest"
	case LatestBlockNumber:
		return "latest"
	case PendingBlockNumber:
		return "pending"
	case SafeBlockNumber:
		return "safe"
	case FinalizedBlockNumber:
		return "finalized"
	case LatestExecutedBlockNumber:
		return "latestExecuted"
	}

	if base == 16 {
		return "0x" + strconv.FormatUint(bn.Uint64(), base)
	}

	return strconv.FormatUint(bn.Uint64(), base)
}

func AsBlockNumber(no interface{}) BlockNumber {
	switch no := no.(type) {
	case *big.Int:
		return BlockNumber(no.Int64())
	case BlockNumber:
		return no
	case *BlockNumber:
		return *no
	case int:
		return BlockNumber(no)
	case int64:
		return BlockNumber(no)
	case uint64:
		return BlockNumber(no)
	case string:
		var bn BlockNumber
		if err := json.Unmarshal([]byte(strconv.Quote(no)), &bn); err == nil {
			return bn
		}
	case fmt.Stringer:
		var bn BlockNumber
		if err := json.Unmarshal([]byte(strconv.Quote(no.String())), &bn); err == nil {
			return bn
		}
	}

	return LatestExecutedBlockNumber - 1
}

type BlockNumberOrHash struct {
	BlockNumber      *BlockNumber    `json:"blockNumber,omitempty"`
	BlockHash        *libcommon.Hash `json:"blockHash,omitempty"`
	RequireCanonical bool            `json:"requireCanonical,omitempty"`
}

func (bnh *BlockNumberOrHash) UnmarshalJSON(data []byte) error {
	type erased BlockNumberOrHash
	e := erased{}
	err := json.Unmarshal(data, &e)
	if err == nil {
		if e.BlockNumber != nil && e.BlockHash != nil {
			return fmt.Errorf("cannot specify both BlockHash and BlockNumber, choose one or the other")
		}
		if e.BlockNumber == nil && e.BlockHash == nil {
			return fmt.Errorf("at least one of BlockNumber or BlockHash is needed if a dictionary is provided")
		}
		bnh.BlockNumber = e.BlockNumber
		bnh.BlockHash = e.BlockHash
		bnh.RequireCanonical = e.RequireCanonical
		return nil
	}
	// Try simple number first
	blckNum, err := strconv.ParseUint(string(data), 10, 64)
	if err == nil {
		if blckNum > math.MaxInt64 {
			return fmt.Errorf("blocknumber too high")
		}
		bn := BlockNumber(blckNum)
		bnh.BlockNumber = &bn
		return nil
	}
	var input string
	if err := json.Unmarshal(data, &input); err != nil {
		return err
	}
	switch input {
	case "earliest":
		bn := EarliestBlockNumber
		bnh.BlockNumber = &bn
		return nil
	case "latest":
		bn := LatestBlockNumber
		bnh.BlockNumber = &bn
		return nil
	case "pending":
		bn := PendingBlockNumber
		bnh.BlockNumber = &bn
		return nil
	case "safe":
		bn := SafeBlockNumber
		bnh.BlockNumber = &bn
		return nil
	case "finalized":
		bn := FinalizedBlockNumber
		bnh.BlockNumber = &bn
		return nil
	default:
		if len(input) == 66 {
			hash := libcommon.Hash{}
			err := hash.UnmarshalText([]byte(input))
			if err != nil {
				return err
			}
			bnh.BlockHash = &hash
			return nil
		} else {
			if blckNum, err = hexutil.DecodeUint64(input); err != nil {
				return err
			}
			if blckNum > math.MaxInt64 {
				return fmt.Errorf("blocknumber too high")
			}
			bn := BlockNumber(blckNum)
			bnh.BlockNumber = &bn
			return nil
		}
	}
}

func (bnh *BlockNumberOrHash) Number() (BlockNumber, bool) {
	if bnh.BlockNumber != nil {
		return *bnh.BlockNumber, true
	}
	return BlockNumber(0), false
}

func (bnh *BlockNumberOrHash) Hash() (libcommon.Hash, bool) {
	if bnh.BlockHash != nil {
		return *bnh.BlockHash, true
	}
	return libcommon.Hash{}, false
}

func BlockNumberOrHashWithNumber(blockNr BlockNumber) BlockNumberOrHash {
	return BlockNumberOrHash{
		BlockNumber:      &blockNr,
		BlockHash:        nil,
		RequireCanonical: false,
	}
}

func BlockNumberOrHashWithHash(hash libcommon.Hash, canonical bool) BlockNumberOrHash {
	return BlockNumberOrHash{
		BlockNumber:      nil,
		BlockHash:        &hash,
		RequireCanonical: canonical,
	}
}

type BlockReference BlockNumberOrHash

func (br *BlockReference) UnmarshalJSON(data []byte) error {
	return ((*BlockNumberOrHash)(br)).UnmarshalJSON(data)
}

func (br BlockReference) Number() (BlockNumber, bool) {
	return ((*BlockNumberOrHash)(&br)).Number()
}

func (br BlockReference) Hash() (libcommon.Hash, bool) {
	return ((*BlockNumberOrHash)(&br)).Hash()
}

func (br BlockReference) String() string {
	if br.BlockNumber != nil {
		return br.BlockNumber.String()
	}

	if br.BlockHash != nil {
		return br.BlockHash.String()
	}

	return ""
}

func AsBlockReference(ref interface{}) BlockReference {
	switch ref := ref.(type) {
	case *big.Int:
		return IntBlockReference(ref)
	case BlockNumber:
		return BlockReference{BlockNumber: &ref}
	case *BlockNumber:
		return BlockReference{BlockNumber: ref}
	case int64:
		bn := BlockNumber(ref)
		return BlockReference{BlockNumber: &bn}
	case uint64:
		return Uint64BlockReference(ref)
	case libcommon.Hash:
		return HashBlockReference(ref)
	case *libcommon.Hash:
		return HashBlockReference(*ref)
	}

	return BlockReference{}
}

func IntBlockReference(blockNr *big.Int) BlockReference {
	if blockNr == nil {
		return BlockReference{}
	}

	bn := BlockNumber(blockNr.Int64())
	return BlockReference{
		BlockNumber:      &bn,
		BlockHash:        nil,
		RequireCanonical: false,
	}
}

func Uint64BlockReference(blockNr uint64) BlockReference {
	bn := BlockNumber(blockNr)
	return BlockReference{
		BlockNumber:      &bn,
		BlockHash:        nil,
		RequireCanonical: false,
	}
}

func HashBlockReference(hash libcommon.Hash, canonical ...bool) BlockReference {
	if len(canonical) == 0 {
		canonical = []bool{false}
	}

	return BlockReference{
		BlockNumber:      nil,
		BlockHash:        &hash,
		RequireCanonical: canonical[0],
	}
}

// DecimalOrHex unmarshals a non-negative decimal or hex parameter into a uint64.
type DecimalOrHex uint64

// UnmarshalJSON implements json.Unmarshaler.
func (dh *DecimalOrHex) UnmarshalJSON(data []byte) error {
	input := strings.TrimSpace(string(data))
	if len(input) >= 2 && input[0] == '"' && input[len(input)-1] == '"' {
		input = input[1 : len(input)-1]
	}

	value, err := strconv.ParseUint(input, 10, 64)
	if err != nil {
		value, err = hexutil.DecodeUint64(input)
	}
	if err != nil {
		return err
	}
	*dh = DecimalOrHex(value)
	return nil
}

func (ts Timestamp) TurnIntoUint64() uint64 {
	return uint64(ts)
}

func (ts *Timestamp) UnmarshalJSON(data []byte) error {
	input := strings.TrimSpace(string(data))
	if len(input) >= 2 && input[0] == '"' && input[len(input)-1] == '"' {
		input = input[1 : len(input)-1]
	}

	// parse string to uint64
	timestamp, err := strconv.ParseUint(input, 10, 64)
	if err != nil {

		// try hex number
		if timestamp, err = hexutil.DecodeUint64(input); err != nil {
			return err
		}

	}

	*ts = Timestamp(timestamp)
	return nil

}
