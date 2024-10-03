package state

import (
	"bytes"
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/c2h5oh/datasize"
	"github.com/erigontech/erigon-lib/common/datadir"
	"github.com/erigontech/erigon-lib/common/dir"
	"github.com/erigontech/erigon-lib/kv"
	"github.com/erigontech/erigon-lib/kv/order"
	"github.com/erigontech/erigon-lib/kv/rawdbv3"
	"github.com/erigontech/erigon-lib/kv/stream"
	"github.com/erigontech/erigon-lib/log/v3"
	"github.com/erigontech/erigon-lib/seg"
)

//Sqeeze: ForeignKeys-aware compression of file

// Sqeeze - re-compress file
// TODO: care of ForeignKeys
func (a *Aggregator) Sqeeze(ctx context.Context, domain kv.Domain) error {
	for _, to := range domainFiles(a.dirs, domain) {
		_, fileName := filepath.Split(to)
		fromStep, toStep, err := ParseStepsFromFileName(fileName)
		if err != nil {
			return err
		}
		if toStep-fromStep < DomainMinStepsToCompress {
			continue
		}

		tempFileCopy := filepath.Join(a.dirs.Tmp, fileName)
		if err := datadir.CopyFile(to, tempFileCopy); err != nil {
			return err
		}

		if err := a.sqeezeDomainFile(ctx, domain, tempFileCopy, to); err != nil {
			return err
		}
		_ = os.Remove(tempFileCopy)
		_ = os.Remove(strings.ReplaceAll(to, ".kv", ".bt"))
		_ = os.Remove(strings.ReplaceAll(to, ".kv", ".bt.torrent"))
		_ = os.Remove(strings.ReplaceAll(to, ".kv", ".kvei"))
		_ = os.Remove(strings.ReplaceAll(to, ".kv", ".kvei.torrent"))
		_ = os.Remove(strings.ReplaceAll(to, ".kv", ".kv.torrent"))
	}
	return nil
}

func (a *Aggregator) sqeezeDomainFile(ctx context.Context, domain kv.Domain, from, to string) error {
	if domain == kv.CommitmentDomain {
		panic("please use SqueezeCommitmentFiles func")
	}

	compression := a.d[domain].compression
	compressCfg := a.d[domain].compressCfg

	a.logger.Info("[sqeeze] file", "f", to, "cfg", compressCfg, "c", compression)
	decompressor, err := seg.NewDecompressor(from)
	if err != nil {
		return err
	}
	defer decompressor.Close()
	defer decompressor.EnableReadAhead().DisableReadAhead()
	r := seg.NewReader(decompressor.MakeGetter(), seg.DetectCompressType(decompressor.MakeGetter()))

	c, err := seg.NewCompressor(ctx, "sqeeze", to, a.dirs.Tmp, compressCfg, log.LvlInfo, a.logger)
	if err != nil {
		return err
	}
	defer c.Close()
	w := seg.NewWriter(c, compression)
	if err := w.ReadFrom(r); err != nil {
		return err
	}
	if err := c.Compress(); err != nil {
		return err
	}

	return nil
}

// SqueezeCommitmentFiles should be called only when NO EXECUTION is running.
// Removes commitment files and suppose following aggregator shutdown and restart  (to integrate new files and rebuild indexes)
func (ac *AggregatorRoTx) SqueezeCommitmentFiles(mergedAgg *AggregatorRoTx) error {
	if !ac.a.commitmentValuesTransform {
		return nil
	}

	commitment := ac.d[kv.CommitmentDomain]
	accounts := ac.d[kv.AccountsDomain]
	storage := ac.d[kv.StorageDomain]

	// oh, again accessing domain.files directly, again and again..
	mergedAccountFiles := mergedAgg.d[kv.AccountsDomain].d.dirtyFiles.Items()
	mergedStorageFiles := mergedAgg.d[kv.StorageDomain].d.dirtyFiles.Items()
	mergedCommitFiles := mergedAgg.d[kv.CommitmentDomain].d.dirtyFiles.Items()

	for _, f := range accounts.files {
		f.src.decompressor.EnableMadvNormal()
	}
	for _, f := range mergedAccountFiles {
		f.decompressor.EnableMadvNormal()
	}
	for _, f := range storage.files {
		f.src.decompressor.EnableMadvNormal()
	}
	for _, f := range mergedStorageFiles {
		f.decompressor.EnableMadvNormal()
	}
	for _, f := range commitment.files {
		f.src.decompressor.EnableMadvNormal()
	}
	for _, f := range mergedCommitFiles {
		f.decompressor.EnableMadvNormal()
	}
	defer func() {
		for _, f := range accounts.files {
			f.src.decompressor.DisableReadAhead()
		}
		for _, f := range mergedAccountFiles {
			f.decompressor.DisableReadAhead()
		}
		for _, f := range storage.files {
			f.src.decompressor.DisableReadAhead()
		}
		for _, f := range mergedStorageFiles {
			f.decompressor.DisableReadAhead()
		}
		for _, f := range commitment.files {
			f.src.decompressor.DisableReadAhead()
		}
		for _, f := range mergedCommitFiles {
			f.decompressor.DisableReadAhead()
		}
	}()

	log.Info("[sqeeze_migration] see target files", "acc", len(mergedAccountFiles), "st", len(mergedStorageFiles), "com", len(mergedCommitFiles))

	getSizeDelta := func(a, b string) (datasize.ByteSize, float32, error) {
		ai, err := os.Stat(a)
		if err != nil {
			return 0, 0, err
		}
		bi, err := os.Stat(b)
		if err != nil {
			return 0, 0, err
		}
		return datasize.ByteSize(ai.Size()) - datasize.ByteSize(bi.Size()), 100.0 * (float32(ai.Size()-bi.Size()) / float32(ai.Size())), nil
	}

	var (
		obsoleteFiles  []string
		temporalFiles  []string
		processedFiles int
		ai, si         int
		sizeDelta      = datasize.B
		sqExt          = ".squeezed"
	)
	logEvery := time.NewTicker(30 * time.Second)
	defer logEvery.Stop()

	for ci := 0; ci < len(mergedCommitFiles); ci++ {
		cf := mergedCommitFiles[ci]
		for ai = 0; ai < len(mergedAccountFiles); ai++ {
			if mergedAccountFiles[ai].startTxNum == cf.startTxNum && mergedAccountFiles[ai].endTxNum == cf.endTxNum {
				break
			}
		}
		for si = 0; si < len(mergedStorageFiles); si++ {
			if mergedStorageFiles[si].startTxNum == cf.startTxNum && mergedStorageFiles[si].endTxNum == cf.endTxNum {
				break
			}
		}
		if ai == len(mergedAccountFiles) || si == len(mergedStorageFiles) {
			ac.a.logger.Info("[sqeeze_migration] commitment file has no corresponding account or storage file", "commitment", cf.decompressor.FileName())
			continue
		}

		err := func() error {
			af, sf := mergedAccountFiles[ai], mergedStorageFiles[si]

			steps := cf.endTxNum/ac.a.aggregationStep - cf.startTxNum/ac.a.aggregationStep
			compression := commitment.d.compression
			if steps < DomainMinStepsToCompress {
				compression = seg.CompressNone
			}
			ac.a.logger.Info("[sqeeze_migration] file start", "original", cf.decompressor.FileName(),
				"progress", fmt.Sprintf("%d/%d", ci+1, len(mergedAccountFiles)), "compress_cfg", commitment.d.compressCfg, "compress", compression)

			originalPath := cf.decompressor.FilePath()
			squeezedTmpPath := originalPath + sqExt + ".tmp"

			squeezedCompr, err := seg.NewCompressor(context.Background(), "squeeze", squeezedTmpPath, ac.a.dirs.Tmp,
				commitment.d.compressCfg, log.LvlInfo, commitment.d.logger)
			if err != nil {
				return err
			}
			defer squeezedCompr.Close()

			reader := seg.NewReader(cf.decompressor.MakeGetter(), compression)
			reader.Reset(0)

			writer := seg.NewWriter(squeezedCompr, commitment.d.compression)
			rng := MergeRange{needMerge: true, from: af.startTxNum, to: af.endTxNum}
			vt, err := commitment.commitmentValTransformDomain(rng, accounts, storage, af, sf)
			if err != nil {
				return fmt.Errorf("failed to create commitment value transformer: %w", err)
			}

			i := 0
			var k, v []byte
			for reader.HasNext() {
				k, _ = reader.Next(k[:0])
				v, _ = reader.Next(v[:0])
				i += 2

				if k == nil {
					// nil keys are not supported for domains
					continue
				}

				if !bytes.Equal(k, keyCommitmentState) {
					v, err = vt(v, af.startTxNum, af.endTxNum)
					if err != nil {
						return fmt.Errorf("failed to transform commitment value: %w", err)
					}
				}
				if err = writer.AddWord(k); err != nil {
					return fmt.Errorf("write key word: %w", err)
				}
				if err = writer.AddWord(v); err != nil {
					return fmt.Errorf("write value word: %w", err)
				}

				select {
				case <-logEvery.C:
					ac.a.logger.Info("[sqeeze_migration]", "file", cf.decompressor.FileName(), "k", fmt.Sprintf("%x", k),
						"progress", fmt.Sprintf("%d/%d", i, cf.decompressor.Count()))
				default:
				}
			}

			if err = writer.Compress(); err != nil {
				return err
			}
			writer.Close()

			squeezedPath := originalPath + sqExt
			if err = os.Rename(squeezedTmpPath, squeezedPath); err != nil {
				return err
			}
			temporalFiles = append(temporalFiles, squeezedPath)

			delta, deltaP, err := getSizeDelta(originalPath, squeezedPath)
			if err != nil {
				return err
			}
			sizeDelta += delta

			ac.a.logger.Info("[sqeeze_migration] file done", "original", filepath.Base(originalPath),
				"sizeDelta", fmt.Sprintf("%s (%.1f%%)", delta.HR(), deltaP))

			fromStep, toStep := af.startTxNum/ac.a.StepSize(), af.endTxNum/ac.a.StepSize()

			// need to remove all indexes for commitment file as well
			obsoleteFiles = append(obsoleteFiles,
				originalPath,
				commitment.d.kvBtFilePath(fromStep, toStep),
				commitment.d.kvAccessorFilePath(fromStep, toStep),
				commitment.d.kvExistenceIdxFilePath(fromStep, toStep),
			)
			processedFiles++
			return nil
		}()
		if err != nil {
			return fmt.Errorf("failed to squeeze commitment file %q: %w", cf.decompressor.FileName(), err)
		}
	}

	ac.a.logger.Info("[squeeze_migration] squeezed files has been produced, removing obsolete files",
		"toRemove", len(obsoleteFiles), "processed", fmt.Sprintf("%d/%d", processedFiles, len(mergedCommitFiles)))
	for _, path := range obsoleteFiles {
		if err := os.Remove(path); err != nil && !errors.Is(err, os.ErrNotExist) {
			return err
		}
		ac.a.logger.Debug("[sqeeze_migration] obsolete file removal", "path", path)
	}
	ac.a.logger.Info("[sqeeze_migration] indices removed, renaming temporal files ")

	for _, path := range temporalFiles {
		if err := os.Rename(path, strings.TrimSuffix(path, sqExt)); err != nil {
			return err
		}
		ac.a.logger.Debug("[sqeeze_migration] temporal file renaming", "path", path)
	}
	ac.a.logger.Info("[sqeeze_migration] done", "sizeDelta", sizeDelta.HR(), "files", len(mergedAccountFiles))

	return nil
}

func (a *Aggregator) RebuildCommitmentFiles(ctx context.Context, roTx kv.RwTx, txNumsReader *rawdbv3.TxNumsReader, domains *SharedDomains) ([]byte, error) {
	// roTx, err := a.db.BeginRo(ctx)
	// if err != nil {
	// 	return nil, err
	// }

	ac, ok := roTx.(HasAggTx).AggTx().(*AggregatorRoTx)
	if !ok {
		return nil, errors.New("failed to get state aggregator")
	}

	fileRanges := domains.FileRanges()
	start := time.Now()

	defer func() {
		a.logger.Info("Commitment rebuild finished", "duration", time.Since(start))
	}()

	for i, r := range fileRanges[kv.AccountsDomain] {
		a.logger.Info("scanning", "range", r.String("", domains.StepSize()), "shards", fmt.Sprintf("%d/%d", i+1, len(fileRanges[kv.AccountsDomain])))

		fromTxNumRange, toTxNumRange := r.FromTo()
		ok, blockNum, err := txNumsReader.FindBlockNum(roTx, toTxNumRange-1)
		if err != nil {
			a.logger.Warn("Failed to find block number for txNum", "txNum", toTxNumRange, "err", err)
			return nil, err
		}
		_ = ok
		domains.SetBlockNum(blockNum)
		domains.SetTxNum(toTxNumRange - 1)
		fffs := ac.Files()
		a.logger.Info("files before rebuild", "count", len(fffs), "files", fffs)
		a.logger.Info("setting numbers", "block", domains.BlockNum(), "txNum", domains.TxNum())

		rh, err := domains.ComputeCommitment(ctx, false, 0, "")
		if err != nil {
			return nil, err
		}
		a.logger.Info("Initialised", "root", fmt.Sprintf("%x", rh))

		it, err := ac.DomainRangeAsOf(kv.AccountsDomain, int(fromTxNumRange), math.MaxInt, order.Asc, roTx)
		if err != nil {
			return nil, err
		}
		defer it.Close()

		itS, err := ac.DomainRangeAsOf(kv.StorageDomain, int(fromTxNumRange), math.MaxInt, order.Asc, roTx)
		if err != nil {
			return nil, err
		}
		defer itS.Close()
		// itC, err := ac.DomainRangeAsOf(kv.CodeDomain, int(fromTxNumRange), math.MaxInt, order.Asc, roTx)
		// if err != nil {
		// 	return nil, err
		// }
		// defer itC.Close()

		keyIter := stream.UnionKV(it, itS, -1)
		// keyIter := stream.UnionKV(stream.UnionKV(it, itS, -1), itC, -1) //

		rebuiltCommit, err := domains.RebuildCommitmentRange(ctx, keyIter, blockNum, fromTxNumRange, toTxNumRange)
		if err != nil {
			return nil, err
		}
		_ = rebuiltCommit

		keyIter.Close()
		it.Close()
		itS.Close()

		a.recalcVisibleFiles(a.dirtyFilesEndTxNumMinimax())
		// a.recalcVisibleFilesMinimaxTxNum()

		// ac.Close()
		// roTx.Rollback()

		// roTx, err = a.db.BeginRo(ctx)
		// if err != nil {
		// 	return nil, err
		// }
		// defer roTx.Rollback()

		ac.Close()
		ac = a.BeginFilesRo()
		defer ac.Close()

		domains.SetAggTx(ac)

		// ac, ok := roTx.(HasAggTx).AggTx().(*AggregatorRoTx)
		// if !ok {
		// 	panic("type AggTx need AggTx method")
		// }
		domains.ClearRam(false) //
		// domains.SetTx(roTx)

		fffs = ac.Files()
		a.logger.Info("files after rebuild", "count", len(fffs), "files", fffs)

		// a.d[kv.CommitmentDomain].DumpStepRangeOnDisk(ctx, rebuiltCommit.StepFrom, rebuiltCommit.StepTo, domains.StepSize(), nil)

		// sd.aggTx.d[kv.CommitmentDomain].Close()
		// newComTx := comDom.BeginFilesRo()
		// sd.aggTx.d[kv.CommitmentDomain] = newComTx
		// sd.aggTx.a.visibleFilesMinimaxTxNum.Store(sd.aggTx.minimaxTxNumInDomainFiles())

		// sd.aggTx.a.recalcVisibleFiles(uint64(to))

		// domains.Close()
		// ac.Close()			 //
		// // ac.RestrictSubsetFileDeletions(false)
		// // if err = cfg.agg.MergeLoop(ctx); err != nil {
		// // 	return nil, err
		// // }
		// if err = rwTx.Commit(); err != nil {
		// 	return nil, err
		// }

		// rwTx, err = cfg.db.BeginRw(ctx)
		// if err != nil {
		// 	return nil, err
		// }

		// domains, err = state.NewSharedDomains(rwTx, log.New())
		// if err != nil {
		// 	return nil, err
		// }
		// ac, ok = rwTx.(state.HasAggTx).AggTx().(*state.AggregatorRoTx)
		// if !ok {
		// 	panic(fmt.Errorf("type %T need AggTx method", rwTx))
		// }
		// domains.SetTx(rwTx)
		// ac = cfg.agg.BeginFilesRo()
		// defer ac.Close()
		// domains.SetAggTx(ac)

		a.logger.Info("range finished", "hash", hex.EncodeToString(rebuiltCommit.RootHash), "range", r.String("", domains.StepSize()), "block", blockNum)
	}
	roTx.Rollback()
	ac.Close()

	return nil, nil
}

func domainFiles(dirs datadir.Dirs, domain kv.Domain) []string {
	files, err := dir.ListFiles(dirs.SnapDomain, ".kv")
	if err != nil {
		panic(err)
	}
	res := make([]string, 0, len(files))
	for _, f := range files {
		if !strings.Contains(f, domain.String()) {
			continue
		}
		res = append(res, f)
	}
	return res
}
