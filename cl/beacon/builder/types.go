package builder

import (
	"encoding/json"
	"math/big"

	"github.com/ledgerwatch/erigon-lib/common"
	"github.com/ledgerwatch/erigon/cl/cltypes"
	"github.com/ledgerwatch/erigon/cl/cltypes/solid"
	"github.com/ledgerwatch/log/v3"
)

type ExecutionPayloadHeader struct {
	Version string `json:"version"`
	Data    struct {
		Message struct {
			Header             *cltypes.Eth1Header                    `json:"header"`
			BlobKzgCommitments *solid.ListSSZ[*cltypes.KZGCommitment] `json:"blob_kzg_commitments"`
			Value              string                                 `json:"value"`
			PubKey             common.Bytes48                         `json:"pubkey"`
		} `json:"message"`
		Signature common.Bytes96 `json:"signature"`
	} `json:"data"`
}

func (h ExecutionPayloadHeader) BlockValue() *big.Int {
	if h.Data.Message.Value == "" {
		return nil
	}
	//blockValue := binary.LittleEndian.Uint64([]byte(h.Data.Message.Value))
	blockValue, ok := new(big.Int).SetString(h.Data.Message.Value, 10)
	if !ok {
		log.Warn("cannot parse block value", "value", h.Data.Message.Value)
	}
	return blockValue
}

type BlindedBlockResponse struct {
	Version string          `json:"version"`
	Data    json.RawMessage `json:"data"`
}
