package remotedbserver

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"

	"github.com/ledgerwatch/turbo-geth/common"
	"github.com/ledgerwatch/turbo-geth/core"
	"github.com/ledgerwatch/turbo-geth/core/types"
	"github.com/ledgerwatch/turbo-geth/ethdb/remote"
	"github.com/ledgerwatch/turbo-geth/log"
	"github.com/ledgerwatch/turbo-geth/rlp"
)

type EthBackendServer struct {
	remote.UnimplementedETHBACKENDServer // must be embedded to have forward compatible implementations.

	eth    core.Backend
	events *Events
}

func NewEthBackendServer(eth core.Backend, events *Events) *EthBackendServer {
	return &EthBackendServer{eth: eth, events: events}
}

func (s *EthBackendServer) Add(_ context.Context, in *remote.TxRequest) (*remote.AddReply, error) {
	signedTx := new(types.Transaction)
	out := &remote.AddReply{Hash: common.Hash{}.Bytes()}

	if err := rlp.DecodeBytes(in.Signedtx, signedTx); err != nil {
		return out, err
	}

	if err := s.eth.TxPool().AddLocal(signedTx); err != nil {
		return out, err
	}

	out.Hash = signedTx.Hash().Bytes()
	return out, nil
}

func (s *EthBackendServer) Etherbase(_ context.Context, _ *remote.EtherbaseRequest) (*remote.EtherbaseReply, error) {
	out := &remote.EtherbaseReply{Hash: common.Hash{}.Bytes()}

	base, err := s.eth.Etherbase()
	if err != nil {
		return out, err
	}

	out.Hash = base.Hash().Bytes()
	return out, nil
}

func (s *EthBackendServer) NetVersion(_ context.Context, _ *remote.NetVersionRequest) (*remote.NetVersionReply, error) {
	id, err := s.eth.NetVersion()
	if err != nil {
		return &remote.NetVersionReply{}, err
	}
	return &remote.NetVersionReply{Id: id}, nil
}

func (s *EthBackendServer) Subscribe(_ *remote.SubscribeRequest, subscribeServer remote.ETHBACKEND_SubscribeServer) error {
	fmt.Println("establishing subscription channel with RPC daemon")
	wg := sync.WaitGroup{}
	wg.Add(1)

	s.events.AddHeaderSubscription(func(h *types.Header) error {
		fmt.Println("sending header", h.Number)

		payload, err := json.Marshal(h)
		if err != nil {
			log.Warn("error while marshaling a header", "err", err)
			return err
		}

		err = subscribeServer.Send(&remote.SubscribeReply{Data: payload})
		if err != nil {
			wg.Done()
		}
		return err
	})
	fmt.Println("subscription channel established with RPC daemon")
	wg.Wait()
	fmt.Println("subscription channel closed with RPC daemon")
	return nil
}
