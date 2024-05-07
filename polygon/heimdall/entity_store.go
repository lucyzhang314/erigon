package heimdall

import (
	"context"
	"encoding/binary"
	"encoding/json"

	"github.com/ledgerwatch/erigon-lib/kv"
	"github.com/ledgerwatch/erigon/turbo/services"
)

type entity interface {
	Checkpoint | Milestone | Span
}

type genericEntityStore[TEntity entity] struct {
	tx    kv.RwTx
	table string

	getLastEntityId func(ctx context.Context, tx kv.Tx) (uint64, bool, error)
	loadEntityBytes func(ctx context.Context, tx kv.Getter, id uint64) ([]byte, error)
}

func newCheckpointStore(tx kv.RwTx, reader services.BorCheckpointReader) entityStore {
	return newGenericEntityStore[Checkpoint](tx, kv.BorCheckpoints, reader.LastCheckpointId, reader.Checkpoint)
}

func newMilestoneStore(tx kv.RwTx, reader services.BorMilestoneReader) entityStore {
	return newGenericEntityStore[Milestone](tx, kv.BorMilestones, reader.LastMilestoneId, reader.Milestone)
}

func newSpanStore(tx kv.RwTx, reader services.BorSpanReader) entityStore {
	return newGenericEntityStore[Span](tx, kv.BorSpans, reader.LastSpanId, reader.Span)
}

func newGenericEntityStore[TEntity entity](
	tx kv.RwTx,
	table string,
	getLastEntityId func(ctx context.Context, tx kv.Tx) (uint64, bool, error),
	loadEntityBytes func(ctx context.Context, tx kv.Getter, id uint64) ([]byte, error),
) entityStore {
	return &genericEntityStore[TEntity]{
		tx:    tx,
		table: table,

		getLastEntityId: getLastEntityId,
		loadEntityBytes: loadEntityBytes,
	}
}

func (s *genericEntityStore[TEntity]) GetLastEntityId(ctx context.Context) (uint64, bool, error) {
	return s.getLastEntityId(ctx, s.tx)
}

func (s *genericEntityStore[TEntity]) GetEntity(ctx context.Context, id uint64) (any, error) {
	jsonBytes, err := s.loadEntityBytes(ctx, s.tx, id)
	if err != nil {
		return nil, err
	}

	var e TEntity
	if err := json.Unmarshal(jsonBytes, &e); err != nil {
		return nil, err
	}

	return &e, nil
}

func (s *genericEntityStore[TEntity]) PutEntity(_ context.Context, id uint64, entity any) error {
	jsonBytes, err := json.Marshal(entity)
	if err != nil {
		return err
	}

	var idBytes [8]byte
	binary.BigEndian.PutUint64(idBytes[:], id)

	return s.tx.Put(s.table, idBytes[:], jsonBytes)
}
