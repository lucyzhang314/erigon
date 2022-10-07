package handlers

import (
	"context"
	"testing"
	"time"

	"github.com/ledgerwatch/erigon/cmd/lightclient/rpc/lightrpc"
	"github.com/ledgerwatch/erigon/cmd/lightclient/sentinel/communication/p2p"
	"github.com/ledgerwatch/erigon/cmd/lightclient/sentinel/communication/ssz_snappy"
	"github.com/ledgerwatch/erigon/cmd/lightclient/sentinel/peers"
	"github.com/ledgerwatch/erigon/common"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/protocol"
	basichost "github.com/libp2p/go-libp2p/p2p/host/basic"
	swarmt "github.com/libp2p/go-libp2p/p2p/net/swarm/testing"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func initializeNetwork(t *testing.T, ctx context.Context) (*ConsensusHandlers, host.Host, host.Host) {
	h1, err := basichost.NewHost(swarmt.GenSwarm(t), nil)
	require.NoError(t, err)

	h2, err := basichost.NewHost(swarmt.GenSwarm(t), nil)
	require.NoError(t, err)
	h2pi := h2.Peerstore().PeerInfo(h2.ID())
	require.NoError(t, h1.Connect(ctx, h2pi))

	return NewConsensusHandlers(h2, &peers.Peers{}, &lightrpc.MetadataV1{}), h1, h2
}

func TestPingHandler(t *testing.T) {
	ctx := context.TODO()

	handlers, h1, h2 := initializeNetwork(t, ctx)
	defer h1.Close()
	defer h2.Close()
	handlers.Start()

	stream, err := h1.NewStream(ctx, h2.ID(), protocol.ID(PingProtocolV1))
	require.NoError(t, err)
	packet := &p2p.Ping{
		Id: 32,
	}
	codec := ssz_snappy.NewStreamCodec(stream)
	_, err = codec.WritePacket(packet)
	require.NoError(t, err)
	require.NoError(t, codec.CloseWriter())
	time.Sleep(100 * time.Millisecond)
	r := &p2p.Ping{}

	code := make([]byte, 1)
	stream.Read(code)
	assert.Equal(t, code, []byte{SuccessfullResponsePrefix})

	_, err = codec.Decode(r)
	require.NoError(t, err)

	assert.Equal(t, r, packet)
}

func TestStatusHandler(t *testing.T) {
	ctx := context.TODO()

	handlers, h1, h2 := initializeNetwork(t, ctx)
	defer h1.Close()
	defer h2.Close()
	handlers.Start()

	stream, err := h1.NewStream(ctx, h2.ID(), protocol.ID(StatusProtocolV1))
	require.NoError(t, err)
	packet := &p2p.Status{
		ForkDigest:    common.Hex2Bytes("69696969"),
		HeadRoot:      make([]byte, 32),
		FinalizedRoot: make([]byte, 32),
		HeadSlot:      666999,
	}
	codec := ssz_snappy.NewStreamCodec(stream)
	_, err = codec.WritePacket(packet)
	require.NoError(t, err)
	require.NoError(t, codec.CloseWriter())
	time.Sleep(100 * time.Millisecond)
	r := &p2p.Status{}

	code := make([]byte, 1)
	stream.Read(code)
	assert.Equal(t, code, []byte{SuccessfullResponsePrefix})

	_, err = codec.Decode(r)
	require.NoError(t, err)

	assert.Equal(t, r, packet)
}
