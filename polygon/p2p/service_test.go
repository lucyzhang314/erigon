package p2p

import (
	"context"
	"errors"
	"fmt"
	"math/big"
	"sync"
	"testing"
	"time"

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
	"github.com/ledgerwatch/erigon/rlp"
	"github.com/ledgerwatch/erigon/turbo/testlog"
)

func newServiceTest(t *testing.T, requestIdGenerator RequestIdGenerator) *serviceTest {
	ctx, cancel := context.WithCancel(context.Background())
	ctrl := gomock.NewController(t)
	logger := testlog.Logger(t, log.LvlTrace)
	sentryClient := direct.NewMockSentryClient(ctrl)
	return &serviceTest{
		ctx:                         ctx,
		ctxCancel:                   cancel,
		t:                           t,
		sentryClient:                sentryClient,
		service:                     newService(100, logger, sentryClient, requestIdGenerator),
		headersRequestResponseMocks: map[uint64]requestResponseMock{},
	}
}

type serviceTest struct {
	ctx                           context.Context
	ctxCancel                     context.CancelFunc
	t                             *testing.T
	sentryClient                  *direct.MockSentryClient
	service                       Service
	headersRequestResponseMocksMu sync.Mutex
	headersRequestResponseMocks   map[uint64]requestResponseMock
	peerEvents                    chan *sentry.PeerEvent
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
func (st *serviceTest) run(f func(ctx context.Context, t *testing.T)) {
	st.t.Run("start", func(_ *testing.T) {
		st.service.Start(st.ctx)
	})

	st.t.Run("test", func(t *testing.T) {
		f(st.ctx, t)
	})

	st.t.Run("stop", func(_ *testing.T) {
		st.ctxCancel()
		st.service.Stop()
	})
}

func (st *serviceTest) mockExpectPenalizePeer(peerId PeerId) {
	st.sentryClient.
		EXPECT().
		PenalizePeer(gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, req *sentry.PenalizePeerRequest, _ ...grpc.CallOption) (*emptypb.Empty, error) {
			if peerId.H512() != req.PeerId {
				return nil, fmt.Errorf("peerId != req.PeerId - %v vs %v", peerId.H512(), req.PeerId)
			}

			return &emptypb.Empty{}, nil
		}).
		Times(1)
}

func (st *serviceTest) mockSentryStreams(mocks ...requestResponseMock) {
	// default mocks
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

	st.mockSentryInboundMessagesStream(mocks...)
	st.mockSentryPeerEventsStream()
}

func (st *serviceTest) mockSentryInboundMessagesStream(mocks ...requestResponseMock) {
	st.headersRequestResponseMocksMu.Lock()
	defer st.headersRequestResponseMocksMu.Unlock()
	for _, mock := range mocks {
		st.headersRequestResponseMocks[mock.requestId] = mock
	}

	inboundMessageStreamChan := make(chan *sentry.InboundMessage, len(st.headersRequestResponseMocks))
	mockSentryInboundMessagesStream := &mockSentryMessagesStream[*sentry.InboundMessage]{
		ctx:    st.ctx,
		stream: inboundMessageStreamChan,
	}

	st.sentryClient.
		EXPECT().
		Messages(gomock.Any(), gomock.Any(), gomock.Any()).
		Return(mockSentryInboundMessagesStream, nil).
		AnyTimes()
	st.sentryClient.
		EXPECT().
		SendMessageById(gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, req *sentry.SendMessageByIdRequest, _ ...grpc.CallOption) (*sentry.SentPeers, error) {
			if sentry.MessageId_GET_BLOCK_HEADERS_66 != req.Data.Id {
				return nil, fmt.Errorf("MessageId_GET_BLOCK_HEADERS_66 != req.Data.Id - %v", req.Data.Id)
			}

			var pkt eth.GetBlockHeadersPacket66
			if err := rlp.DecodeBytes(req.Data.Data, &pkt); err != nil {
				return nil, err
			}

			st.headersRequestResponseMocksMu.Lock()
			defer st.headersRequestResponseMocksMu.Unlock()
			mock, ok := st.headersRequestResponseMocks[pkt.RequestId]
			if !ok {
				return nil, fmt.Errorf("unexpected request id: %d", pkt.RequestId)
			}

			reqPeerId := PeerIdFromH512(req.PeerId)
			if mock.wantRequestPeerId != reqPeerId {
				return nil, fmt.Errorf("wantRequestPeerId != reqPeerId - %v vs %v", mock.wantRequestPeerId, reqPeerId)
			}

			if mock.wantRequestOriginNumber != pkt.Origin.Number {
				return nil, fmt.Errorf("wantRequestOriginNumber != pkt.Origin.Number - %v vs %v", mock.wantRequestOriginNumber, pkt.Origin.Number)
			}

			if mock.wantRequestAmount != pkt.Amount {
				return nil, fmt.Errorf("wantRequestAmount != pkt.Amount - %v vs %v", mock.wantRequestAmount, pkt.Amount)
			}

			for _, inboundMessage := range mock.mockResponseInboundMessages {
				inboundMessageStreamChan <- inboundMessage
			}

			return nil, nil
		}).
		AnyTimes()
}

func (st *serviceTest) mockSentryPeerEventsStream() {
	peerConnectEvents := []*sentry.PeerEvent{
		{
			EventId: sentry.PeerEvent_Connect,
			PeerId:  PeerIdFromUint64(1).H512(),
		},
		{
			EventId: sentry.PeerEvent_Connect,
			PeerId:  PeerIdFromUint64(2).H512(),
		},
	}

	streamChan := make(chan *sentry.PeerEvent, len(peerConnectEvents))
	for _, event := range peerConnectEvents {
		streamChan <- event
	}

	st.peerEvents = streamChan
	st.sentryClient.
		EXPECT().
		PeerEvents(gomock.Any(), gomock.Any(), gomock.Any()).
		Return(&mockSentryMessagesStream[*sentry.PeerEvent]{
			ctx:    st.ctx,
			stream: streamChan,
		}, nil).
		AnyTimes()
}

func (st *serviceTest) mockDisconnectPeerEvent(peerId PeerId) {
	st.peerEvents <- &sentry.PeerEvent{
		EventId: sentry.PeerEvent_Disconnect,
		PeerId:  peerId.H512(),
	}
}

type requestResponseMock struct {
	requestId                   uint64
	mockResponseInboundMessages []*sentry.InboundMessage
	wantRequestPeerId           PeerId
	wantRequestOriginNumber     uint64
	wantRequestAmount           uint64
}

type mockSentryMessagesStream[M any] struct {
	ctx    context.Context
	stream <-chan M
}

func (s *mockSentryMessagesStream[M]) Recv() (M, error) {
	var nilValue M
	return nilValue, nil
}

func (s *mockSentryMessagesStream[M]) Header() (metadata.MD, error) {
	return nil, nil
}

func (s *mockSentryMessagesStream[M]) Trailer() metadata.MD {
	return nil
}

func (s *mockSentryMessagesStream[M]) CloseSend() error {
	return nil
}

func (s *mockSentryMessagesStream[M]) Context() context.Context {
	return context.Background()
}

func (s *mockSentryMessagesStream[M]) SendMsg(_ any) error {
	return nil
}

func (s *mockSentryMessagesStream[M]) RecvMsg(msg any) error {
	select {
	case <-s.ctx.Done():
		return s.ctx.Err()
	case mockMsg := <-s.stream:
		switch any(mockMsg).(type) {
		case *sentry.InboundMessage:
			msg, ok := msg.(*sentry.InboundMessage)
			if !ok {
				return errors.New("unexpected msg type")
			}

			mockMsg := any(mockMsg).(*sentry.InboundMessage)
			msg.Id = mockMsg.Id
			msg.Data = mockMsg.Data
			msg.PeerId = mockMsg.PeerId
		case *sentry.PeerEvent:
			msg, ok := msg.(*sentry.PeerEvent)
			if !ok {
				return errors.New("unexpected msg type")
			}

			mockMsg := any(mockMsg).(*sentry.PeerEvent)
			msg.PeerId = mockMsg.PeerId
			msg.EventId = mockMsg.EventId
		default:
			return errors.New("unsupported type")
		}

		return nil
	}
}

func newMockRequestGenerator(requestIds ...uint64) RequestIdGenerator {
	var idx int
	idxPtr := &idx
	return func() uint64 {
		if *idxPtr >= len(requestIds) {
			panic("mock request generator does not have any request ids left")
		}

		res := requestIds[*idxPtr]
		*idxPtr++
		return res
	}
}

func newMockBlockHeadersPacket66Bytes(t *testing.T, requestId uint64, numHeaders int) []byte {
	headers := make([]*types.Header, numHeaders)
	for i := range headers {
		headers[i] = &types.Header{
			Number: big.NewInt(int64(i) + 1),
		}
	}

	blockHeadersPacket66 := eth.BlockHeadersPacket66{
		RequestId:          requestId,
		BlockHeadersPacket: headers,
	}
	blockHeadersPacket66Bytes, err := rlp.EncodeToBytes(&blockHeadersPacket66)
	require.NoError(t, err)
	return blockHeadersPacket66Bytes
}

func TestServiceFetchHeaders(t *testing.T) {
	t.Parallel()

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
			Data:   newMockBlockHeadersPacket66Bytes(t, requestId*2, 2),
		},
		{
			Id:     sentry.MessageId_BLOCK_HEADERS_66,
			PeerId: peerId.H512(),
			Data:   newMockBlockHeadersPacket66Bytes(t, requestId, 2),
		},
	}
	mockRequestResponse := requestResponseMock{
		requestId:                   requestId,
		mockResponseInboundMessages: mockInboundMessages,
		wantRequestPeerId:           peerId,
		wantRequestOriginNumber:     1,
		wantRequestAmount:           2,
	}

	test := newServiceTest(t, newMockRequestGenerator(requestId))
	test.mockSentryStreams(mockRequestResponse)
	test.run(func(ctx context.Context, t *testing.T) {
		headers, err := test.service.FetchHeaders(ctx, 1, 3, peerId)
		require.NoError(t, err)
		require.Len(t, headers, 2)
		require.Equal(t, uint64(1), headers[0].Number.Uint64())
		require.Equal(t, uint64(2), headers[1].Number.Uint64())
	})
}

func TestServiceErrInvalidFetchHeadersRange(t *testing.T) {
	t.Parallel()

	test := newServiceTest(t, newMockRequestGenerator(1))
	test.mockSentryStreams()
	test.run(func(ctx context.Context, t *testing.T) {
		headers, err := test.service.FetchHeaders(ctx, 3, 1, PeerIdFromUint64(1))
		var errInvalidFetchHeadersRange *ErrInvalidFetchHeadersRange
		require.ErrorAs(t, err, &errInvalidFetchHeadersRange)
		require.Equal(t, uint64(3), errInvalidFetchHeadersRange.start)
		require.Equal(t, uint64(1), errInvalidFetchHeadersRange.end)
		require.Nil(t, headers)
	})
}

func TestServiceFetchHeadersShouldPenalizePeerWhenErrInvalidRlpErr(t *testing.T) {
	t.Parallel()

	peerId := PeerIdFromUint64(1)
	requestId := uint64(1234)
	mockInboundMessages := []*sentry.InboundMessage{
		{
			Id:     sentry.MessageId_BLOCK_HEADERS_66,
			PeerId: peerId.H512(),
			Data:   []byte{'i', 'n', 'v', 'a', 'l', 'i', 'd', '.', 'r', 'l', 'p'},
		},
	}
	mockRequestResponse := requestResponseMock{
		requestId:                   requestId,
		mockResponseInboundMessages: mockInboundMessages,
		wantRequestPeerId:           peerId,
		wantRequestOriginNumber:     1,
		wantRequestAmount:           2,
	}

	test := newServiceTest(t, newMockRequestGenerator(requestId))
	test.mockSentryStreams(mockRequestResponse)
	test.mockExpectPenalizePeer(peerId)
	test.run(func(ctx context.Context, t *testing.T) {
		headers, err := test.service.FetchHeaders(ctx, 1, 3, peerId)
		require.Error(t, err)
		require.Nil(t, headers)
	})
}

func TestListPeersMayHaveBlockNum(t *testing.T) {
	t.Parallel()

	peerId1 := PeerIdFromUint64(1)
	requestId1 := uint64(1234)
	mockInboundMessages1 := []*sentry.InboundMessage{
		{
			Id:     sentry.MessageId_BLOCK_HEADERS_66,
			PeerId: peerId1.H512(),
			Data:   newMockBlockHeadersPacket66Bytes(t, requestId1, 2),
		},
	}
	mockRequestResponse1 := requestResponseMock{
		requestId:                   requestId1,
		mockResponseInboundMessages: mockInboundMessages1,
		wantRequestPeerId:           peerId1,
		wantRequestOriginNumber:     1,
		wantRequestAmount:           2,
	}
	requestId2 := uint64(1235)
	mockInboundMessages2 := []*sentry.InboundMessage{
		{
			Id:     sentry.MessageId_BLOCK_HEADERS_66,
			PeerId: peerId1.H512(),
			// peer returns 0 headers for requestId2 - peer does not have this header range
			Data: newMockBlockHeadersPacket66Bytes(t, requestId2, 0),
		},
	}
	mockRequestResponse2 := requestResponseMock{
		requestId:                   requestId2,
		mockResponseInboundMessages: mockInboundMessages2,
		wantRequestPeerId:           peerId1,
		wantRequestOriginNumber:     3,
		wantRequestAmount:           2,
	}

	test := newServiceTest(t, newMockRequestGenerator(requestId1, requestId2))
	test.mockSentryStreams(mockRequestResponse1, mockRequestResponse2)
	test.run(func(ctx context.Context, t *testing.T) {
		var peerIds []PeerId // peers which may have blocks 1 and 2
		require.Eventuallyf(t, func() bool {
			peerIds = test.service.ListPeersMayHaveBlockNum(2)
			return len(peerIds) == 2
		}, time.Second, 100*time.Millisecond, "expected number of initial peers never satisfied: want=2, have=%d", len(peerIds))

		headers, err := test.service.FetchHeaders(ctx, 1, 3, peerId1) // fetch headers 1 and 2
		require.NoError(t, err)
		require.Len(t, headers, 2)
		require.Equal(t, uint64(1), headers[0].Number.Uint64())
		require.Equal(t, uint64(2), headers[1].Number.Uint64())

		peerIds = test.service.ListPeersMayHaveBlockNum(4) // peers which may have blocks 1,2,3,4
		require.Len(t, peerIds, 2)

		var errIncompleteFetchHeadersResponse *ErrIncompleteFetchHeadersResponse
		headers, err = test.service.FetchHeaders(ctx, 3, 5, peerId1) // fetch headers 3 and 4
		require.ErrorAs(t, err, &errIncompleteFetchHeadersResponse)  // peer 1 does not have headers 3 and 4
		require.Equal(t, uint64(3), errIncompleteFetchHeadersResponse.requestStart)
		require.Equal(t, uint64(5), errIncompleteFetchHeadersResponse.requestEnd)
		require.Equal(t, uint64(0), errIncompleteFetchHeadersResponse.first)
		require.Equal(t, uint64(0), errIncompleteFetchHeadersResponse.last)
		require.Equal(t, 0, errIncompleteFetchHeadersResponse.amount)
		require.Equal(t, uint64(3), errIncompleteFetchHeadersResponse.LowestMissingBlockNum())
		require.Nil(t, headers)

		// should be one peer less now given that we know that peer 1 does not have block num 4
		peerIds = test.service.ListPeersMayHaveBlockNum(4)
		require.Len(t, peerIds, 1)
	})
}

func TestListPeersMayHaveBlockNumDoesNotReturnPeerIdAfterDisconnect(t *testing.T) {
	t.Parallel()

	peerId := PeerIdFromUint64(1)
	test := newServiceTest(t, newMockRequestGenerator())
	test.mockSentryStreams()
	test.run(func(ctx context.Context, t *testing.T) {
		wantPeerCount := 2

		var peerIds []PeerId
		require.Eventuallyf(t, func() bool {
			peerIds = test.service.ListPeersMayHaveBlockNum(2)
			return len(peerIds) == 2
		}, time.Second, 100*time.Millisecond, "expected number of peers never satisfied: want=%d, have=%d", wantPeerCount, len(peerIds))

		test.mockDisconnectPeerEvent(peerId)

		require.Eventuallyf(t, func() bool {
			peerIds = test.service.ListPeersMayHaveBlockNum(2)
			return len(peerIds) == 1
		}, time.Second, 100*time.Millisecond, "expected number of peers never satisfied: want=%d, have=%d", wantPeerCount, len(peerIds))

		require.Equal(t, PeerIdFromUint64(2), peerIds[0])
	})
}
