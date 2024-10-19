// Copyright 2024 The Erigon Authors
// This file is part of Erigon.
//
// Erigon is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// Erigon is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with Erigon. If not, see <http://www.gnu.org/licenses/>.

package heimdall

import (
	"bytes"
	"context"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"path/filepath"
	"runtime"
	"time"

	"github.com/erigontech/erigon-lib/chain"
	"github.com/erigontech/erigon-lib/chain/networkname"
	"github.com/erigontech/erigon-lib/chain/snapcfg"
	"github.com/erigontech/erigon-lib/common"
	"github.com/erigontech/erigon-lib/common/background"
	"github.com/erigontech/erigon-lib/common/dbg"
	"github.com/erigontech/erigon-lib/common/hexutility"
	"github.com/erigontech/erigon-lib/common/length"
	"github.com/erigontech/erigon-lib/downloader/snaptype"
	"github.com/erigontech/erigon-lib/kv"
	"github.com/erigontech/erigon-lib/log/v3"
	"github.com/erigontech/erigon-lib/recsplit"
	"github.com/erigontech/erigon-lib/seg"
	"github.com/erigontech/erigon/core/rawdb"
	coresnaptype "github.com/erigontech/erigon/core/snaptype"
	bortypes "github.com/erigontech/erigon/polygon/bor/types"
)

func init() {
	initTypes()
}

func initTypes() {
	borTypes := append(coresnaptype.BlockSnapshotTypes, SnapshotTypes()...)
	borTypes = append(borTypes, coresnaptype.E3StateTypes...)

	snapcfg.RegisterKnownTypes(networkname.AmoyChainName, borTypes)
	snapcfg.RegisterKnownTypes(networkname.BorMainnetChainName, borTypes)
}

var Enums = struct {
	snaptype.Enums
	Events,
	Spans,
	Checkpoints,
	Milestones snaptype.Enum
}{
	Enums:       snaptype.Enums{},
	Events:      snaptype.MinBorEnum,
	Spans:       snaptype.MinBorEnum + 1,
	Checkpoints: snaptype.MinBorEnum + 2,
	Milestones:  snaptype.MinBorEnum + 3,
}

var Indexes = struct {
	BorTxnHash,
	BorSpanId,
	BorCheckpointId,
	BorMilestoneId snaptype.Index
}{
	BorTxnHash:      snaptype.Index{Name: "borevents"},
	BorSpanId:       snaptype.Index{Name: "borspans"},
	BorCheckpointId: snaptype.Index{Name: "borcheckpoints"},
	BorMilestoneId:  snaptype.Index{Name: "bormilestones"},
}

var (
	Events = snaptype.RegisterType(
		Enums.Events,
		"borevents",
		snaptype.Versions{
			Current:      1, //2,
			MinSupported: 1,
		},
		snaptype.RangeExtractorFunc(
			func(ctx context.Context, blockFrom, blockTo uint64, firstEventId snaptype.FirstKeyGetter, db kv.RoDB, chainConfig *chain.Config, collect func([]byte) error, workers int, lvl log.Lvl, logger log.Logger) (uint64, error) {
				logEvery := time.NewTicker(20 * time.Second)
				defer logEvery.Stop()

				from := hexutility.EncodeTs(blockFrom)
				startEventId := firstEventId(ctx)
				var lastEventId uint64

				if err := kv.BigChunks(db, kv.BorEventNums, from, func(tx kv.Tx, blockNumBytes, eventIdBytes []byte) (bool, error) {
					endEventId := binary.BigEndian.Uint64(eventIdBytes) + 1
					blockNum := binary.BigEndian.Uint64(blockNumBytes)
					blockHash, e := rawdb.ReadCanonicalHash(tx, blockNum)
					if e != nil {
						return false, e
					}

					if blockNum >= blockTo {
						return false, nil
					}

					if e := extractEventRange(startEventId, endEventId, tx, blockNum, blockHash, collect); e != nil {
						return false, e
					}
					startEventId = endEventId

					lastEventId = binary.BigEndian.Uint64(eventIdBytes)
					select {
					case <-ctx.Done():
						return false, ctx.Err()
					case <-logEvery.C:
						var m runtime.MemStats
						if lvl >= log.LvlInfo {
							dbg.ReadMemStats(&m)
						}
						logger.Log(lvl, "[bor snapshots] Dumping bor events", "block num", blockNum,
							"alloc", common.ByteCount(m.Alloc), "sys", common.ByteCount(m.Sys),
						)
					default:
					}
					return true, nil
				}); err != nil {
					return 0, err
				}

				return lastEventId, nil
			}),
		[]snaptype.Index{Indexes.BorTxnHash},
		snaptype.IndexBuilderFunc(
			func(ctx context.Context, sn snaptype.FileInfo, salt uint32, chainConfig *chain.Config, tmpDir string, p *background.Progress, lvl log.Lvl, logger log.Logger) (err error) {
				defer func() {
					if rec := recover(); rec != nil {
						err = fmt.Errorf("BorEventsIdx: at=%d-%d, %v, %s", sn.From, sn.To, rec, dbg.Stack())
					}
				}()
				// Calculate how many records there will be in the index
				d, err := seg.NewDecompressor(sn.Path)
				if err != nil {
					return err
				}
				defer d.Close()
				g := d.MakeGetter()
				var blockNumBuf [length.BlockNum]byte
				var first bool = true
				word := make([]byte, 0, 4096)
				var blockCount int
				var baseEventId uint64
				for g.HasNext() {
					word, _ = g.Next(word[:0])
					if first || !bytes.Equal(blockNumBuf[:], word[length.Hash:length.Hash+length.BlockNum]) {
						blockCount++
						copy(blockNumBuf[:], word[length.Hash:length.Hash+length.BlockNum])
					}
					if first {
						baseEventId = binary.BigEndian.Uint64(word[length.Hash+length.BlockNum : length.Hash+length.BlockNum+8])
						first = false
					}
					select {
					case <-ctx.Done():
						return ctx.Err()
					default:
					}
				}

				rs, err := recsplit.NewRecSplit(recsplit.RecSplitArgs{
					KeyCount:   blockCount,
					Enums:      blockCount > 0,
					BucketSize: 2000,
					LeafSize:   8,
					TmpDir:     tmpDir,
					IndexFile:  filepath.Join(sn.Dir(), snaptype.IdxFileName(sn.Version, sn.From, sn.To, Enums.Events.String())),
					BaseDataID: baseEventId,
				}, logger)
				if err != nil {
					return err
				}
				rs.LogLvl(log.LvlDebug)

				defer d.EnableReadAhead().DisableReadAhead()

				for {
					g.Reset(0)
					first = true
					var i, offset, nextPos uint64
					for g.HasNext() {
						word, nextPos = g.Next(word[:0])
						i++
						if first || !bytes.Equal(blockNumBuf[:], word[length.Hash:length.Hash+length.BlockNum]) {
							if err = rs.AddKey(word[:length.Hash], offset); err != nil {
								return err
							}
							copy(blockNumBuf[:], word[length.Hash:length.Hash+length.BlockNum])
						}
						if first {
							first = false
						}
						select {
						case <-ctx.Done():
							return ctx.Err()
						default:
						}
						offset = nextPos
					}
					if err = rs.Build(ctx); err != nil {
						if errors.Is(err, recsplit.ErrCollision) {
							logger.Info("Building recsplit. Collision happened. It's ok. Restarting with another salt...", "err", err)
							rs.ResetNextSalt()
							continue
						}
						return err
					}

					return nil
				}
			}))

	Spans = snaptype.RegisterType(
		Enums.Spans,
		"borspans",
		snaptype.Versions{
			Current:      1, //2,
			MinSupported: 1,
		},
		snaptype.RangeExtractorFunc(
			func(ctx context.Context, blockFrom, blockTo uint64, firstKeyGetter snaptype.FirstKeyGetter, db kv.RoDB, _ *chain.Config, collect func([]byte) error, workers int, lvl log.Lvl, logger log.Logger) (uint64, error) {
				spanFrom := uint64(SpanIdAt(blockFrom))
				spanTo := uint64(SpanIdAt(blockTo))
				return extractValueRange(ctx, kv.BorSpans, spanFrom, spanTo, db, collect, workers, lvl, logger)
			}),
		[]snaptype.Index{Indexes.BorSpanId},
		snaptype.IndexBuilderFunc(
			func(ctx context.Context, sn snaptype.FileInfo, salt uint32, _ *chain.Config, tmpDir string, p *background.Progress, lvl log.Lvl, logger log.Logger) (err error) {
				d, err := seg.NewDecompressor(sn.Path)

				if err != nil {
					return err
				}
				defer d.Close()

				baseSpanId := uint64(SpanIdAt(sn.From))

				return buildValueIndex(ctx, sn, salt, d, baseSpanId, tmpDir, p, lvl, logger)
			}),
	)

	Checkpoints = snaptype.RegisterType(
		Enums.Checkpoints,
		"borcheckpoints",
		snaptype.Versions{
			Current:      1, //2,
			MinSupported: 1,
		},
		snaptype.RangeExtractorFunc(
			func(ctx context.Context, blockFrom, blockTo uint64, firstKeyGetter snaptype.FirstKeyGetter, db kv.RoDB, _ *chain.Config, collect func([]byte) error, workers int, lvl log.Lvl, logger log.Logger) (uint64, error) {
				var checkpointTo, checkpointFrom CheckpointId

				checkpointId := func(rangeIndex RangeIndex, blockNum uint64) (CheckpointId, error) {
					checkpointId, _, err := rangeIndex.Lookup(ctx, blockNum)
					return CheckpointId(checkpointId), err
				}

				err := db.View(ctx, func(tx kv.Tx) (err error) {
					rangeIndex := NewTxRangeIndex(db, kv.BorCheckpointEnds, tx)

					checkpointFrom, err = checkpointId(rangeIndex, blockFrom)

					//checkpointFrom, err = CheckpointIdAt(tx, blockFrom)

					if err != nil {
						return err
					}

					checkpointTo, err = checkpointId(rangeIndex, blockTo)
					//checkpointTo, err = CheckpointIdAt(tx, blockTo)

					if err != nil {
						return err
					}

					if blockFrom > 0 {
						if prevTo, err := checkpointId(rangeIndex, blockFrom-1); err == nil {
							if prevTo == CheckpointId(checkpointFrom) {
								if prevTo == CheckpointId(checkpointTo) {
									checkpointFrom = 0
									checkpointTo = 0
								} else {
									checkpointFrom++
								}
							}
						}
					}

					return err
				})

				if err != nil {
					return 0, err
				}

				return extractValueRange(ctx, kv.BorCheckpoints, uint64(checkpointFrom), uint64(checkpointTo), db, collect, workers, lvl, logger)
			}),
		[]snaptype.Index{Indexes.BorCheckpointId},
		snaptype.IndexBuilderFunc(
			func(ctx context.Context, sn snaptype.FileInfo, salt uint32, _ *chain.Config, tmpDir string, p *background.Progress, lvl log.Lvl, logger log.Logger) (err error) {
				d, err := seg.NewDecompressor(sn.Path)

				if err != nil {
					return err
				}
				defer d.Close()

				gg := d.MakeGetter()

				var firstCheckpointId uint64

				if gg.HasNext() {
					buf, _ := d.MakeGetter().Next(nil)
					var firstCheckpoint Checkpoint

					if err = json.Unmarshal(buf, &firstCheckpoint); err != nil {
						return err
					}

					firstCheckpointId = uint64(firstCheckpoint.Id)
				}

				return buildValueIndex(ctx, sn, salt, d, firstCheckpointId, tmpDir, p, lvl, logger)
			}),
	)

	Milestones = snaptype.RegisterType(
		Enums.Milestones,
		"bormilestones",
		snaptype.Versions{
			Current:      1, //2,
			MinSupported: 1,
		},
		snaptype.RangeExtractorFunc(
			func(ctx context.Context, blockFrom, blockTo uint64, firstKeyGetter snaptype.FirstKeyGetter, db kv.RoDB, _ *chain.Config, collect func([]byte) error, workers int, lvl log.Lvl, logger log.Logger) (uint64, error) {
				var milestoneFrom, milestoneTo MilestoneId

				milestoneId := func(rangeIndex RangeIndex, blockNum uint64) (MilestoneId, error) {
					milestoneId, _, err := rangeIndex.Lookup(ctx, blockNum)
					return MilestoneId(milestoneId), err
				}
				err := db.View(ctx, func(tx kv.Tx) (err error) {
					rangeIndex := NewTxRangeIndex(db, kv.BorMilestoneEnds, tx)

					milestoneFrom, err = milestoneId(rangeIndex, blockFrom)

					if err != nil && !errors.Is(err, ErrMilestoneNotFound) {
						return err
					}

					milestoneTo, err = milestoneId(rangeIndex, blockTo)

					if err != nil && !errors.Is(err, ErrMilestoneNotFound) {
						return err
					}

					if milestoneFrom > 0 && blockFrom > 0 {
						if prevTo, err := milestoneId(rangeIndex, blockFrom-1); err == nil && prevTo == milestoneFrom {
							if prevTo == milestoneFrom {
								if prevTo == milestoneTo {
									milestoneFrom = 0
									milestoneTo = 0
								} else {
									milestoneFrom++
								}
							}
						}
					}

					return nil
				})

				if err != nil {
					return 0, err
				}

				return extractValueRange(ctx, kv.BorMilestones, uint64(milestoneFrom), uint64(milestoneTo), db, collect, workers, lvl, logger)
			}),
		[]snaptype.Index{Indexes.BorMilestoneId},
		snaptype.IndexBuilderFunc(
			func(ctx context.Context, sn snaptype.FileInfo, salt uint32, _ *chain.Config, tmpDir string, p *background.Progress, lvl log.Lvl, logger log.Logger) (err error) {
				d, err := seg.NewDecompressor(sn.Path)

				if err != nil {
					return err
				}
				defer d.Close()

				gg := d.MakeGetter()

				var firstMilestoneId uint64

				if gg.HasNext() {
					buf, _ := gg.Next(nil)
					if len(buf) > 0 {
						var firstMilestone Milestone
						if err = json.Unmarshal(buf, &firstMilestone); err != nil {
							return err
						}
						firstMilestoneId = uint64(firstMilestone.Id)
					}
				}

				return buildValueIndex(ctx, sn, salt, d, firstMilestoneId, tmpDir, p, lvl, logger)
			}),
	)
)

var recordWaypoints bool

func RecordWayPoints(value bool) {
	recordWaypoints = value
	initTypes()
}

func SnapshotTypes() []snaptype.Type {
	if recordWaypoints {
		return []snaptype.Type{Events, Spans, Checkpoints, Milestones}
	}

	return []snaptype.Type{Events, Spans}
}

func CheckpointsEnabled() bool {
	for _, snapType := range SnapshotTypes() {
		if snapType.Enum() == Checkpoints.Enum() {
			return true
		}
	}

	return false
}

func MilestonesEnabled() bool {
	for _, snapType := range SnapshotTypes() {
		if snapType.Enum() == Milestones.Enum() {
			return true
		}
	}

	return false
}

func extractValueRange(ctx context.Context, table string, valueFrom, valueTo uint64, db kv.RoDB, collect func([]byte) error, workers int, lvl log.Lvl, logger log.Logger) (uint64, error) {
	logEvery := time.NewTicker(20 * time.Second)
	defer logEvery.Stop()

	if err := kv.BigChunks(db, table, hexutility.EncodeTs(valueFrom), func(tx kv.Tx, idBytes, valueBytes []byte) (bool, error) {
		id := binary.BigEndian.Uint64(idBytes)
		if id >= valueTo {
			return false, nil
		}
		if e := collect(valueBytes); e != nil {
			return false, e
		}
		select {
		case <-ctx.Done():
			return false, ctx.Err()
		case <-logEvery.C:
			var m runtime.MemStats
			if lvl >= log.LvlInfo {
				dbg.ReadMemStats(&m)
			}
			logger.Log(lvl, "[bor snapshots] Dumping bor values", "id", id,
				"alloc", common.ByteCount(m.Alloc), "sys", common.ByteCount(m.Sys),
			)
		default:
		}
		return true, nil
	}); err != nil {
		return valueTo, err
	}
	return valueTo, nil
}

func buildValueIndex(ctx context.Context, sn snaptype.FileInfo, salt uint32, d *seg.Decompressor, baseId uint64, tmpDir string, p *background.Progress, lvl log.Lvl, logger log.Logger) (err error) {
	defer func() {
		if rec := recover(); rec != nil {
			err = fmt.Errorf("BorSpansIdx: at=%d-%d, %v, %s", sn.From, sn.To, rec, dbg.Stack())
		}
	}()

	rs, err := recsplit.NewRecSplit(recsplit.RecSplitArgs{
		KeyCount:   d.Count(),
		Enums:      d.Count() > 0,
		BucketSize: 2000,
		LeafSize:   8,
		TmpDir:     tmpDir,
		IndexFile:  filepath.Join(sn.Dir(), sn.Type.IdxFileName(sn.Version, sn.From, sn.To)),
		BaseDataID: baseId,
		Salt:       &salt,
	}, logger)
	if err != nil {
		return err
	}
	rs.LogLvl(log.LvlDebug)

	defer d.EnableReadAhead().DisableReadAhead()

	for {
		g := d.MakeGetter()
		var i, offset, nextPos uint64
		var key [8]byte
		for g.HasNext() {
			nextPos, _ = g.Skip()
			binary.BigEndian.PutUint64(key[:], i)
			i++
			if err = rs.AddKey(key[:], offset); err != nil {
				return err
			}
			select {
			case <-ctx.Done():
				return ctx.Err()
			default:
			}
			offset = nextPos
		}
		if err = rs.Build(ctx); err != nil {
			if errors.Is(err, recsplit.ErrCollision) {
				logger.Info("Building recsplit. Collision happened. It's ok. Restarting with another salt...", "err", err)
				rs.ResetNextSalt()
				continue
			}
			return err
		}

		return nil
	}
}

// extractEventRange moves [startEventID, endEventID) to snapshots
func extractEventRange(startEventId, endEventId uint64, tx kv.Tx, blockNum uint64, blockHash common.Hash, collect func([]byte) error) error {
	var blockNumBuf [8]byte
	var eventIdBuf [8]byte
	txnHash := bortypes.ComputeBorTxHash(blockNum, blockHash)
	binary.BigEndian.PutUint64(blockNumBuf[:], blockNum)
	for eventId := startEventId; eventId < endEventId; eventId++ {
		binary.BigEndian.PutUint64(eventIdBuf[:], eventId)
		event, err := tx.GetOne(kv.BorEvents, eventIdBuf[:])
		if err != nil {
			return err
		}
		snapshotRecord := make([]byte, len(event)+length.Hash+length.BlockNum+8)
		copy(snapshotRecord, txnHash[:])
		copy(snapshotRecord[length.Hash:], blockNumBuf[:])
		binary.BigEndian.PutUint64(snapshotRecord[length.Hash+length.BlockNum:], eventId)
		copy(snapshotRecord[length.Hash+length.BlockNum+8:], event)
		if err := collect(snapshotRecord); err != nil {
			return err
		}
	}
	return nil
}