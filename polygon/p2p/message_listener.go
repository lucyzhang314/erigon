package p2p

import (
	"context"
	"sync"

	"github.com/ledgerwatch/log/v3"
	"google.golang.org/grpc"

	"github.com/ledgerwatch/erigon-lib/direct"
	"github.com/ledgerwatch/erigon-lib/gointerfaces/sentry"
	sentrymulticlient "github.com/ledgerwatch/erigon/p2p/sentry/sentry_multi_client"
)

type MessageListener interface {
	Start(ctx context.Context)
	Stop()
	RegisterBlockHeaders66(observer messageObserver[*sentry.InboundMessage])
	UnregisterBlockHeaders66(observer messageObserver[*sentry.InboundMessage])
	RegisterPeerEventObserver(observer messageObserver[*sentry.PeerEvent])
	UnregisterPeerEventObserver(observer messageObserver[*sentry.PeerEvent])
}

func NewMessageListener(logger log.Logger, sentryClient direct.SentryClient) MessageListener {
	return &messageListener{
		logger:                  logger,
		sentryClient:            sentryClient,
		inboundMessageObservers: map[sentry.MessageId]map[messageObserver[*sentry.InboundMessage]]struct{}{},
	}
}

type messageListener struct {
	once                      sync.Once
	streamCtx                 context.Context
	streamCtxCancel           context.CancelFunc
	logger                    log.Logger
	sentryClient              direct.SentryClient
	inboundMessageObserversMu sync.Mutex
	inboundMessageObservers   map[sentry.MessageId]map[messageObserver[*sentry.InboundMessage]]struct{}
	stopWg                    sync.WaitGroup
}

func (ml *messageListener) Start(ctx context.Context) {
	ml.once.Do(func() {
		ml.streamCtx, ml.streamCtxCancel = context.WithCancel(ctx)
		go ml.listenBlockHeaders66()
	})
}

func (ml *messageListener) Stop() {
	ml.streamCtxCancel()
	ml.stopWg.Wait()
}

func (ml *messageListener) RegisterBlockHeaders66(observer messageObserver[*sentry.InboundMessage]) {
	ml.register(observer, sentry.MessageId_BLOCK_HEADERS_66)
}

func (ml *messageListener) UnregisterBlockHeaders66(observer messageObserver[*sentry.InboundMessage]) {
	ml.unregister(observer, sentry.MessageId_BLOCK_HEADERS_66)
}

func (ml *messageListener) RegisterPeerEventObserver(observer messageObserver[*sentry.PeerEvent]) {
	// TODO
}

func (ml *messageListener) UnregisterPeerEventObserver(observer messageObserver[*sentry.PeerEvent]) {
	// TODO
}

func (ml *messageListener) register(observer messageObserver[*sentry.InboundMessage], messageId sentry.MessageId) {
	ml.inboundMessageObserversMu.Lock()
	defer ml.inboundMessageObserversMu.Unlock()

	if observers, ok := ml.inboundMessageObservers[messageId]; ok {
		observers[observer] = struct{}{}
	} else {
		ml.inboundMessageObservers[messageId] = map[messageObserver[*sentry.InboundMessage]]struct{}{
			observer: {},
		}
	}
}

func (ml *messageListener) unregister(observer messageObserver[*sentry.InboundMessage], messageId sentry.MessageId) {
	ml.inboundMessageObserversMu.Lock()
	defer ml.inboundMessageObserversMu.Unlock()

	if observers, ok := ml.inboundMessageObservers[messageId]; ok {
		delete(observers, observer)
	}
}

func (ml *messageListener) listenBlockHeaders66() {
	ml.listenInboundMessage("BlockHeaders66", sentry.MessageId_BLOCK_HEADERS_66)
}

func (ml *messageListener) listenInboundMessage(name string, msgId sentry.MessageId) {
	ml.stopWg.Add(1)
	defer ml.stopWg.Done()

	sentrymulticlient.SentryReconnectAndPumpStreamLoop(
		ml.streamCtx,
		ml.sentryClient,
		ml.statusDataFactory(),
		name,
		ml.messageStreamFactory([]sentry.MessageId{msgId}),
		ml.inboundMessageFactory(),
		ml.handleInboundMessageHandler(),
		nil,
		ml.logger,
	)
}

func (ml *messageListener) statusDataFactory() sentrymulticlient.StatusDataFactory {
	return func() *sentry.StatusData {
		return &sentry.StatusData{}
	}
}

func (ml *messageListener) messageStreamFactory(ids []sentry.MessageId) sentrymulticlient.SentryMessageStreamFactory {
	return func(streamCtx context.Context, sentryClient direct.SentryClient) (sentrymulticlient.SentryMessageStream, error) {
		return sentryClient.Messages(streamCtx, &sentry.MessagesRequest{Ids: ids}, grpc.WaitForReady(true))
	}
}

func (ml *messageListener) inboundMessageFactory() sentrymulticlient.MessageFactory[*sentry.InboundMessage] {
	return func() *sentry.InboundMessage {
		return new(sentry.InboundMessage)
	}
}

func (ml *messageListener) handleInboundMessageHandler() sentrymulticlient.InboundMessageHandler[*sentry.InboundMessage] {
	return func(_ context.Context, msg *sentry.InboundMessage, _ direct.SentryClient) error {
		ml.notifyInboundMessageObservers(msg)
		return nil
	}
}

func (ml *messageListener) notifyInboundMessageObservers(msg *sentry.InboundMessage) {
	ml.inboundMessageObserversMu.Lock()
	defer ml.inboundMessageObserversMu.Unlock()

	observers, ok := ml.inboundMessageObservers[msg.Id]
	if !ok {
		return
	}

	for observer := range observers {
		go observer.Notify(msg)
	}
}
