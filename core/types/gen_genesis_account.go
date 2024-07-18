// Code generated by github.com/fjl/gencodec. DO NOT EDIT.

package types

import (
	"encoding/json"
	"errors"
	"math/big"

	"github.com/erigontech/erigon-lib/common"
	"github.com/erigontech/erigon-lib/common/hexutility"
	"github.com/erigontech/erigon/common/math"
)

var _ = (*genesisAccountMarshaling)(nil)

// MarshalJSON marshals as JSON.
func (g GenesisAccount) MarshalJSON() ([]byte, error) {
	type GenesisAccount struct {
		Constructor hexutility.Bytes            `json:"constructor,omitempty"`
		Code        hexutility.Bytes            `json:"code,omitempty"`
		Storage     map[storageJSON]storageJSON `json:"storage,omitempty"`
		Balance     *math.HexOrDecimal256       `json:"balance" gencodec:"required"`
		Nonce       math.HexOrDecimal64         `json:"nonce,omitempty"`
		PrivateKey  hexutility.Bytes            `json:"secretKey,omitempty"`
	}
	var enc GenesisAccount
	enc.Constructor = g.Constructor
	enc.Code = g.Code
	if g.Storage != nil {
		enc.Storage = make(map[storageJSON]storageJSON, len(g.Storage))
		for k, v := range g.Storage {
			enc.Storage[storageJSON(k)] = storageJSON(v)
		}
	}
	enc.Balance = (*math.HexOrDecimal256)(g.Balance)
	enc.Nonce = math.HexOrDecimal64(g.Nonce)
	enc.PrivateKey = g.PrivateKey
	return json.Marshal(&enc)
}

// UnmarshalJSON unmarshals from JSON.
func (g *GenesisAccount) UnmarshalJSON(input []byte) error {
	type GenesisAccount struct {
		Constructor *hexutility.Bytes           `json:"constructor,omitempty"`
		Code        *hexutility.Bytes           `json:"code,omitempty"`
		Storage     map[storageJSON]storageJSON `json:"storage,omitempty"`
		Balance     *math.HexOrDecimal256       `json:"balance" gencodec:"required"`
		Nonce       *math.HexOrDecimal64        `json:"nonce,omitempty"`
		PrivateKey  *hexutility.Bytes           `json:"secretKey,omitempty"`
	}
	var dec GenesisAccount
	if err := json.Unmarshal(input, &dec); err != nil {
		return err
	}
	if dec.Constructor != nil {
		g.Constructor = *dec.Constructor
	}
	if dec.Code != nil {
		g.Code = *dec.Code
	}
	if dec.Storage != nil {
		g.Storage = make(map[common.Hash]common.Hash, len(dec.Storage))
		for k, v := range dec.Storage {
			g.Storage[common.Hash(k)] = common.Hash(v)
		}
	}
	if dec.Balance == nil {
		return errors.New("missing required field 'balance' for GenesisAccount")
	}
	g.Balance = (*big.Int)(dec.Balance)
	if dec.Nonce != nil {
		g.Nonce = uint64(*dec.Nonce)
	}
	if dec.PrivateKey != nil {
		g.PrivateKey = *dec.PrivateKey
	}
	return nil
}
