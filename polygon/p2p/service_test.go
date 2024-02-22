package p2p

import (
	"context"
	"errors"
	"fmt"
	"math/big"
	"sync"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/ledgerwatch/log/v3"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
	"google.golang.org/protobuf/types/known/emptypb"

	"github.com/ledgerwatch/erigon-lib/direct"
	"github.com/ledgerwatch/erigon-lib/gointerfaces/sentry"
	"github.com/ledgerwatch/erigon/core/types"
	"github.com/ledgerwatch/erigon/eth/protocols/eth"
	"github.com/ledgerwatch/erigon/p2p"
	"github.com/ledgerwatch/erigon/rlp"
	"github.com/ledgerwatch/erigon/turbo/testlog"
)

func newServiceTest(t *testing.T, requestIdGenerator RequestIdGenerator) *serviceTest {
	ctrl := gomock.NewController(t)
	logger := testlog.Logger(t, log.LvlTrace)
	sentryClient := direct.NewMockSentryClient(ctrl)
	return &serviceTest{
		t:            t,
		sentryClient: sentryClient,
		service:      newService(p2p.Config{}, logger, sentryClient, requestIdGenerator),
	}
}

type serviceTest struct {
	t            *testing.T
	sentryClient *direct.MockSentryClient
	service      Service
}

// run is needed so that we can properly shut down tests involving the p2p service due to how the sentry multi
// client SentryReconnectAndPumpStreamLoop works.
//
// Using t.Cleanup to call service.Stop instead does not work since the mocks generated by gomock cause
// an error when their methods are called after a test has finished - t.Cleanup is run after a
// test has finished, and so we need to make sure that the SentryReconnectAndPumpStreamLoop loop has been stopped
// before the test finishes otherwise we will have flaky tests.
//
// If changing the behaviour here please run "go test -v -count=1000 ./polygon/p2p" and
// "go test -v -count=1 -race ./polygon/p2p" to confirm there are no regressions.
func (st *serviceTest) run(ctx context.Context, f func(t *testing.T)) {
	st.t.Run("start", func(_ *testing.T) {
		st.service.Start(ctx)
	})

	st.t.Run("test", f)

	st.t.Run("stop", func(_ *testing.T) {
		st.service.Stop()
	})
}

func (st *serviceTest) mockExpectPenalizePeer(peerId PeerId) {
	st.sentryClient.
		EXPECT().
		PenalizePeer(gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, req *sentry.PenalizePeerRequest, _ ...grpc.CallOption) (*emptypb.Empty, error) {
			if peerId.H512() != req.PeerId {
				return nil, fmt.Errorf("peerId != reqPeerId - %v vs %v", peerId.H512(), req.PeerId)
			}

			return &emptypb.Empty{}, nil
		}).
		Times(1)
}

func (st *serviceTest) mockSentryBlockHeaders66InboundMessageStream(
	msgs []*sentry.InboundMessage,
	peerId PeerId,
	wantOriginNumber uint64,
	wantAmount uint64,
) {
	var wg sync.WaitGroup
	if len(msgs) > 0 {
		wg.Add(1)
	}

	st.sentryClient.
		EXPECT().
		Messages(gomock.Any(), gomock.Any(), gomock.Any()).
		Return(st.mockSentryStream(&wg, msgs), nil).
		AnyTimes()
	st.sentryClient.
		EXPECT().
		SendMessageById(gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(newSendGetBlockHeaders66MessageMock(&wg, peerId, wantOriginNumber, wantAmount)).
		AnyTimes()
	st.sentryClient.
		EXPECT().
		HandShake(gomock.Any(), gomock.Any(), gomock.Any()).
		Return(nil, nil).
		AnyTimes()
	st.sentryClient.
		EXPECT().
		SetStatus(gomock.Any(), gomock.Any(), gomock.Any()).
		Return(nil, nil).
		AnyTimes()
	st.sentryClient.
		EXPECT().
		MarkDisconnected().
		AnyTimes()
}

func (st *serviceTest) mockSentryStream(wg *sync.WaitGroup, msgs []*sentry.InboundMessage) sentry.Sentry_MessagesClient {
	return &mockSentryMessagesStream{
		wg:   wg,
		msgs: msgs,
	}
}

type mockSentryMessagesStream struct {
	wg   *sync.WaitGroup
	msgs []*sentry.InboundMessage
}

func (s *mockSentryMessagesStream) Recv() (*sentry.InboundMessage, error) {
	return nil, nil
}

func (s *mockSentryMessagesStream) Header() (metadata.MD, error) {
	return nil, nil
}

func (s *mockSentryMessagesStream) Trailer() metadata.MD {
	return nil
}

func (s *mockSentryMessagesStream) CloseSend() error {
	return nil
}

func (s *mockSentryMessagesStream) Context() context.Context {
	return context.Background()
}

func (s *mockSentryMessagesStream) SendMsg(_ any) error {
	return nil
}

func (s *mockSentryMessagesStream) RecvMsg(msg any) error {
	// wait for something external to happen before stream is allowed to produce values
	s.wg.Wait()

	if len(s.msgs) == 0 {
		return nil
	}

	inboundMsg, ok := msg.(*sentry.InboundMessage)
	if !ok {
		return errors.New("unexpected msg type")
	}

	mockMsg := s.msgs[0]
	s.msgs = s.msgs[1:]
	inboundMsg.Id = mockMsg.Id
	inboundMsg.Data = mockMsg.Data
	inboundMsg.PeerId = mockMsg.PeerId
	return nil
}

func newMockRequestGenerator(reqId uint64) RequestIdGenerator {
	return func() uint64 {
		return reqId
	}
}

func newMockBlockHeadersPacket66Bytes(t *testing.T, requestId uint64) []byte {
	blockHeadersPacket66 := eth.BlockHeadersPacket66{
		RequestId: requestId,
		BlockHeadersPacket: []*types.Header{
			{
				Number: big.NewInt(1),
			},
			{
				Number: big.NewInt(2),
			},
			{
				Number: big.NewInt(3),
			},
		},
	}
	blockHeadersPacket66Bytes, err := rlp.EncodeToBytes(&blockHeadersPacket66)
	require.NoError(t, err)
	return blockHeadersPacket66Bytes
}

func newSendGetBlockHeaders66MessageMock(
	wg *sync.WaitGroup,
	wantPeerId PeerId,
	wantOriginNumber uint64,
	wantAmount uint64,
) sendMessageByIdMock {
	return func(_ context.Context, req *sentry.SendMessageByIdRequest, _ ...grpc.CallOption) (*sentry.SentPeers, error) {
		defer wg.Done()

		reqPeerId := PeerIdFromH512(req.PeerId)
		if wantPeerId != reqPeerId {
			return nil, fmt.Errorf("wantPeerId != reqPeerId - %v vs %v", wantPeerId, reqPeerId)
		}

		if sentry.MessageId_GET_BLOCK_HEADERS_66 != req.Data.Id {
			return nil, fmt.Errorf("MessageId_GET_BLOCK_HEADERS_66 != req.Data.Id - %v", req.Data.Id)
		}

		var pkt eth.GetBlockHeadersPacket66
		if err := rlp.DecodeBytes(req.Data.Data, &pkt); err != nil {
			return nil, err
		}

		if wantOriginNumber != pkt.Origin.Number {
			return nil, fmt.Errorf("wantOriginNumber != pkt.Origin.Number - %v vs %v", wantOriginNumber, pkt.Origin.Number)
		}

		if wantAmount != pkt.Amount {
			return nil, fmt.Errorf("wantAmount != pkt.Amount - %v vs %v", wantAmount, pkt.Amount)
		}

		return nil, nil
	}
}

type sendMessageByIdMock func(context.Context, *sentry.SendMessageByIdRequest, ...grpc.CallOption) (*sentry.SentPeers, error)

func TestServiceDownloadHeaders(t *testing.T) {
	ctx := context.Background()
	peerId := PeerIdFromUint64(1)
	requestId := uint64(1234)
	mockInboundMessages := []*sentry.InboundMessage{
		{
			// should get filtered because it is from a different peer id
			PeerId: PeerIdFromUint64(2).H512(),
		},
		{
			// should get filtered because it is for a different msg id
			Id: sentry.MessageId_BLOCK_BODIES_66,
		},
		{
			// should get filtered because it is from a different request id
			Id:     sentry.MessageId_BLOCK_HEADERS_66,
			PeerId: peerId.H512(),
			Data:   newMockBlockHeadersPacket66Bytes(t, requestId*2),
		},
		{
			Id:     sentry.MessageId_BLOCK_HEADERS_66,
			PeerId: peerId.H512(),
			Data:   newMockBlockHeadersPacket66Bytes(t, requestId),
		},
	}

	test := newServiceTest(t, newMockRequestGenerator(requestId))
	test.mockSentryBlockHeaders66InboundMessageStream(mockInboundMessages, peerId, 1, 2)
	test.run(ctx, func(t *testing.T) {
		headers, err := test.service.FetchHeaders(ctx, 1, 3, peerId)
		require.NoError(t, err)
		require.Len(t, headers, 3)
		require.Equal(t, uint64(1), headers[0].Number.Uint64())
		require.Equal(t, uint64(2), headers[1].Number.Uint64())
		require.Equal(t, uint64(3), headers[2].Number.Uint64())
	})
}

func TestServiceInvalidDownloadHeadersRangeErr(t *testing.T) {
	ctx := context.Background()
	test := newServiceTest(t, newMockRequestGenerator(1))
	test.mockSentryBlockHeaders66InboundMessageStream(nil, PeerId{}, 1, 2)
	test.run(ctx, func(t *testing.T) {
		headers, err := test.service.FetchHeaders(ctx, 3, 1, PeerIdFromUint64(1))
		require.ErrorIs(t, err, invalidFetchHeadersRangeErr)
		require.Nil(t, headers)
	})
}

func TestServiceDownloadHeadersShouldPenalizePeerWhenInvalidRlpErr(t *testing.T) {
	ctx := context.Background()
	peerId := PeerIdFromUint64(1)
	requestId := uint64(1234)
	mockInboundMessages := []*sentry.InboundMessage{
		{
			Id:     sentry.MessageId_BLOCK_HEADERS_66,
			PeerId: peerId.H512(),
			Data:   []byte{'i', 'n', 'v', 'a', 'l', 'i', 'd', '.', 'r', 'l', 'p'},
		},
	}

	test := newServiceTest(t, newMockRequestGenerator(requestId))
	test.mockSentryBlockHeaders66InboundMessageStream(mockInboundMessages, peerId, 1, 2)
	test.mockExpectPenalizePeer(peerId)
	test.run(ctx, func(t *testing.T) {
		headers, err := test.service.FetchHeaders(ctx, 1, 3, peerId)
		require.Error(t, err)
		require.Nil(t, headers)
	})
}
