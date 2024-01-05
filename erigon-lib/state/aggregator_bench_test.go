package state

import (
	"bytes"
	"context"
	"fmt"
	"math/rand"
	"os"
	"path"
	"path/filepath"
	"testing"
	"time"

	bloomfilter "github.com/holiman/bloomfilter/v2"
	"github.com/ledgerwatch/erigon-lib/common/datadir"

	"github.com/ledgerwatch/log/v3"
	"github.com/stretchr/testify/require"

	"github.com/ledgerwatch/erigon-lib/common"
	"github.com/ledgerwatch/erigon-lib/common/length"
	"github.com/ledgerwatch/erigon-lib/compress"
	"github.com/ledgerwatch/erigon-lib/kv"
	"github.com/ledgerwatch/erigon-lib/kv/mdbx"
	"github.com/ledgerwatch/erigon-lib/recsplit"
)

func TestName(t *testing.T) {
	m := bloomfilter.OptimalM(10_000_000, 0.01)
	k := bloomfilter.OptimalK(m, 10_000_000)
	fmt.Printf("alex: %x\n", k)
}

func testDbAndAggregatorBench(b *testing.B, aggStep uint64) (kv.RwDB, *AggregatorV3) {
	b.Helper()
	logger := log.New()
	dirs := datadir.New(b.TempDir())
	db := mdbx.NewMDBX(logger).InMem(dirs.Chaindata).WithTableCfg(func(defaultBuckets kv.TableCfg) kv.TableCfg {
		return kv.ChaindataTablesCfg
	}).MustOpen()
	b.Cleanup(db.Close)
	agg, err := NewAggregatorV3(context.Background(), dirs, aggStep, db, logger)
	require.NoError(b, err)
	b.Cleanup(agg.Close)
	return db, agg
}

type txWithCtx struct {
	kv.Tx
	ac *AggregatorV3Context
}

func WrapTxWithCtx(tx kv.Tx, ctx *AggregatorV3Context) *txWithCtx { return &txWithCtx{Tx: tx, ac: ctx} }
func (tx *txWithCtx) AggCtx() interface{}                         { return tx.ac }

func BenchmarkAggregator_Processing(b *testing.B) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	longKeys := queueKeys(ctx, 64, length.Addr+length.Hash)
	vals := queueKeys(ctx, 53, length.Hash)

	aggStep := uint64(100_00)
	db, agg := testDbAndAggregatorBench(b, aggStep)

	tx, err := db.BeginRw(ctx)
	require.NoError(b, err)
	defer func() {
		if tx != nil {
			tx.Rollback()
		}
	}()

	require.NoError(b, err)
	ac := agg.MakeContext()
	defer ac.Close()

	domains := NewSharedDomains(WrapTxWithCtx(tx, ac), log.New())
	defer domains.Close()

	b.ReportAllocs()
	b.ResetTimer()

	var prev []byte
	for i := 0; i < b.N; i++ {
		key := <-longKeys
		val := <-vals
		txNum := uint64(i)
		domains.SetTxNum(txNum)
		err := domains.DomainPut(kv.StorageDomain, key[:length.Addr], key[length.Addr:], val, prev)
		prev = val
		require.NoError(b, err)

		if i%100000 == 0 {
			_, err := domains.ComputeCommitment(ctx, true, domains.BlockNum(), "")
			require.NoError(b, err)
		}
	}
}

func queueKeys(ctx context.Context, seed, ofSize uint64) <-chan []byte {
	rnd := rand.New(rand.NewSource(int64(seed)))
	keys := make(chan []byte, 1)
	go func() {
		for {
			if ctx.Err() != nil {
				break
			}
			bb := make([]byte, ofSize)
			rnd.Read(bb)

			keys <- bb
		}
		close(keys)
	}()
	return keys
}

func Benchmark_BtreeIndex_Allocation(b *testing.B) {
	rnd := rand.New(rand.NewSource(time.Now().UnixNano()))
	for i := 0; i < b.N; i++ {
		now := time.Now()
		count := rnd.Intn(1000000000)
		bt := newBtAlloc(uint64(count), uint64(1<<12), true, nil, nil)
		bt.traverseDfs()
		fmt.Printf("alloc %v\n", time.Since(now))
	}
}

func Benchmark_BtreeIndex_Search(b *testing.B) {
	logger := log.New()
	rnd := rand.New(rand.NewSource(time.Now().UnixNano()))
	tmp := b.TempDir()
	defer os.RemoveAll(tmp)
	dataPath := "../../data/storage.256-288.kv"

	indexPath := path.Join(tmp, filepath.Base(dataPath)+".bti")
	comp := CompressKeys | CompressVals
	err := BuildBtreeIndex(dataPath, indexPath, comp, 1, logger, true)
	require.NoError(b, err)

	M := 1024
	bt, err := OpenBtreeIndex(indexPath, dataPath, uint64(M), comp, false)

	require.NoError(b, err)

	keys, err := pivotKeysFromKV(dataPath)
	require.NoError(b, err)

	for i := 0; i < b.N; i++ {
		p := rnd.Intn(len(keys))
		cur, err := bt.SeekDeprecated(keys[p])
		require.NoErrorf(b, err, "i=%d", i)
		require.EqualValues(b, keys[p], cur.Key())
		require.NotEmptyf(b, cur.Value(), "i=%d", i)
	}

	bt.Close()
}

func benchInitBtreeIndex(b *testing.B, M uint64) (*BtIndex, [][]byte, string) {
	b.Helper()

	logger := log.New()
	tmp := b.TempDir()
	b.Cleanup(func() { os.RemoveAll(tmp) })

	dataPath := generateKV(b, tmp, 52, 10, 1000000, logger, 0)
	indexPath := path.Join(tmp, filepath.Base(dataPath)+".bt")
	bt, err := CreateBtreeIndex(indexPath, dataPath, M, CompressNone, 1, logger, true)
	require.NoError(b, err)

	keys, err := pivotKeysFromKV(dataPath)
	require.NoError(b, err)
	return bt, keys, dataPath
}

func Benchmark_BTree_Seek(b *testing.B) {
	M := uint64(1024)
	bt, keys, _ := benchInitBtreeIndex(b, M)
	defer bt.Close()

	rnd := rand.New(rand.NewSource(time.Now().UnixNano()))

	b.Run("seek_only", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			p := rnd.Intn(len(keys))

			cur, err := bt.SeekDeprecated(keys[p])
			require.NoError(b, err)

			require.EqualValues(b, keys[p], cur.key)
		}
	})

	b.Run("seek_then_next", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			p := rnd.Intn(len(keys))

			cur, err := bt.SeekDeprecated(keys[p])
			require.NoError(b, err)

			require.EqualValues(b, keys[p], cur.key)

			prevKey := common.Copy(keys[p])
			ntimer := time.Duration(0)
			nextKeys := 5000
			for j := 0; j < nextKeys; j++ {
				ntime := time.Now()

				if !cur.Next() {
					break
				}
				ntimer += time.Since(ntime)

				nk := cur.Key()
				if bytes.Compare(prevKey, nk) > 0 {
					b.Fatalf("prev %s cur %s, next key should be greater", prevKey, nk)
				}
				prevKey = nk
			}
			if i%1000 == 0 {
				fmt.Printf("next_access_last[of %d keys] %v\n", nextKeys, ntimer/time.Duration(nextKeys))
			}

		}
	})
}

// requires existing KV index file at ../../data/storage.kv
func Benchmark_Recsplit_Find_ExternalFile(b *testing.B) {
	dataPath := "../../data/storage.kv"
	f, err := os.Stat(dataPath)
	if err != nil || f.IsDir() {
		b.Skip("requires existing KV index file at ../../data/storage.kv")
	}

	rnd := rand.New(rand.NewSource(time.Now().UnixNano()))
	tmp := b.TempDir()

	defer os.RemoveAll(tmp)

	indexPath := dataPath + "i"
	idx, err := recsplit.OpenIndex(indexPath)
	require.NoError(b, err)
	idxr := recsplit.NewIndexReader(idx)

	decomp, err := compress.NewDecompressor(dataPath)
	require.NoError(b, err)
	defer decomp.Close()

	getter := decomp.MakeGetter()

	keys, err := pivotKeysFromKV(dataPath)
	require.NoError(b, err)

	for i := 0; i < b.N; i++ {
		p := rnd.Intn(len(keys))

		offset := idxr.Lookup(keys[p])
		getter.Reset(offset)

		require.True(b, getter.HasNext())

		key, pa := getter.Next(nil)
		require.NotEmpty(b, key)

		value, pb := getter.Next(nil)
		if pb-pa != 1 {
			require.NotEmpty(b, value)
		}

		require.NoErrorf(b, err, "i=%d", i)
		require.EqualValues(b, keys[p], key)
	}
}
