package snapshotsync

import (
	"bytes"
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"path/filepath"
	"time"

	lg "github.com/anacrolix/log"
	"github.com/anacrolix/torrent"
	"github.com/anacrolix/torrent/bencode"
	"github.com/anacrolix/torrent/metainfo"
	"github.com/anacrolix/torrent/storage"
	"github.com/dustin/go-humanize"
	"github.com/ledgerwatch/erigon-lib/gointerfaces/snapshotsync"
	"github.com/ledgerwatch/erigon-lib/kv"
	"github.com/ledgerwatch/erigon/common"
	"github.com/ledgerwatch/erigon/common/debug"
	"github.com/ledgerwatch/erigon/ethdb"
	"github.com/ledgerwatch/log/v3"
)

type Client struct {
	Cli          *torrent.Client
	snapshotsDir string
	trackers     [][]string
}

func New(snapshotsDir string, seeding bool, peerID string) (*Client, error) {
	torrentConfig := DefaultTorrentConfig()
	torrentConfig.Seed = seeding
	torrentConfig.DataDir = snapshotsDir
	torrentConfig.UpnpID = torrentConfig.UpnpID + "leecher"
	torrentConfig.PeerID = peerID
	progressStore, err := storage.NewBoltPieceCompletion(snapshotsDir)
	if err != nil {
		panic(err)
	}
	torrentConfig.DefaultStorage = storage.NewMMapWithCompletion(snapshotsDir, progressStore)

	torrentClient, err := torrent.NewClient(torrentConfig)
	if err != nil {
		log.Error("Fail to start torrnet client", "err", err)
		return nil, fmt.Errorf("fail to start: %w", err)
	}

	{
		if err := BuildTorrentFilesIfNeed(context.Background(), snapshotsDir); err != nil {
			return nil, err
		}

		// add .torrent files
		if err := ForEachTorrentFile(snapshotsDir, func(torrentFileName string) error {
			mi, err := metainfo.LoadFromFile(torrentFileName)
			if err != nil {
				return err
			}
			mi.AnnounceList = Trackers
			//TODO: check hash with hard-coded
			tr, err := torrentClient.AddTorrent(mi)
			if err != nil {
				return err
			}
			tr.AllowDataDownload()
			tr.AllowDataUpload()
			return nil
		}); err != nil {
			return nil, err
		}

		waitForChecksumVerify(context.Background(), torrentClient)
		// add hard-coded hashes as magnet links
		//TODO: add magnets, call torrent.DownloadAll() and wait for <-torrent.GotInfo()
		waitForDownloadAll(context.Background(), torrentClient)

		torrentClient.Close()
		progressStore.Close()
		/*
			mi.InfoBytes, err = bencode.Marshal(info)
			if err != nil {
				log.Fatal(err)
			}
			err := info.BuildFromFilePath(a[])
			if err != nil {
				log.Fatal(err)
			}

			mi.InfoBytes, err = bencode.Marshal(a[0])
			if err != nil {
				panic(err)
			}
			infoHash := mi.HashInfoBytes()
			_ = infoHash // TODO: compare with hardcoded hash
			tr, err := torrentClient.AddTorrent(mi)
			if err != nil {
				panic(err)
			}
		*/
		panic(1)

		/*
			mi := &metainfo.MetaInfo{
				CreationDate: time.Now().Unix(),
				CreatedBy:    "erigon",
				AnnounceList: Trackers,
			}
			infoHash := metainfo.NewHashFromHex("8800e00ccdd575c01198cfab5adb5957100a0baa")
			magnet := mi.Magnet(&infoHash, nil)
			tr, err := torrentClient.AddMagnet(magnet.String())
			if err != nil {
				panic(err)
			}
			go func() {
				// Wait until we get the metainfo, then check for the data.
				<-tr.GotInfo()
				fmt.Printf("got info: %x\n", tr.Info().Pieces)
				//tr.DownloadAll()
				//torrentClient.WaitAll()
				//tr.VerifyData()
			}()

			for i := 0; i < 100; i++ {
				wr := tr.Stats().BytesWritten
				fmt.Printf("alex: %s, %dMb, %d\n", tr.Name(), wr.Int64()/1024/1024, tr.Stats().ActivePeers)
				time.Sleep(2 * time.Second)
			}

			torrentClient.WaitAll()

			ts := &torrent.TorrentSpec{
				Trackers: Trackers,
				InfoHash: infoHash,
				//DisplayName: a[0].Name,
				InfoBytes: mi.InfoBytes,
			}

			_ = ts
			//fmt.Printf("%x, %x\n", infoHash, infoBytes)
		*/

		//fmt.Printf("%s\n", a[0].Files[0].Path, ts.)
		panic(1)
	}

	return &Client{
		Cli:          torrentClient,
		snapshotsDir: snapshotsDir,
		trackers:     Trackers,
	}, nil
}

func DefaultTorrentConfig() *torrent.ClientConfig {
	torrentConfig := torrent.NewDefaultClientConfig()
	torrentConfig.ListenPort = 0
	torrentConfig.NoDHT = true
	torrentConfig.DisableTrackers = false
	torrentConfig.Debug = false
	torrentConfig.Logger = NewAdapterLogger()
	torrentConfig.Logger = torrentConfig.Logger.FilterLevel(lg.Info)
	return torrentConfig
}

func (cli *Client) Torrents() []metainfo.Hash {
	t := cli.Cli.Torrents()
	hashes := make([]metainfo.Hash, 0, len(t))
	for _, v := range t {
		hashes = append(hashes, v.InfoHash())
	}
	return hashes
}
func (cli *Client) Load(tx kv.Tx) error {
	log.Info("Load added torrents")
	return tx.ForEach(kv.SnapshotInfo, []byte{}, func(k, infoHashBytes []byte) error {
		if !bytes.HasPrefix(k[8:], []byte(SnapshotInfoHashPrefix)) {
			return nil
		}
		networkID, snapshotName := ParseInfoHashKey(k)
		infoHash := metainfo.Hash{}
		copy(infoHash[:], infoHashBytes)
		infoBytes, err := tx.GetOne(kv.SnapshotInfo, MakeInfoBytesKey(snapshotName, networkID))
		if err != nil {
			return err
		}

		log.Info("Add torrent", "snapshot", snapshotName, "hash", infoHash.String(), "infobytes", len(infoBytes) > 0)
		_, err = cli.AddTorrentSpec(snapshotName, infoHash, infoBytes)
		if err != nil {
			return err
		}

		return nil
	})
}

func (cli *Client) SavePeerID(db kv.Putter) error {
	return db.Put(kv.BittorrentInfo, []byte(kv.BittorrentPeerID), cli.PeerID())
}

func (cli *Client) Close() {
	cli.Cli.Close()
}

func (cli *Client) PeerID() []byte {
	peerID := cli.Cli.PeerID()
	return peerID[:]
}
func (cli *Client) AddTorrentSpec(snapshotName string, snapshotHash metainfo.Hash, infoBytes []byte) (*torrent.Torrent, error) {
	t, ok := cli.Cli.Torrent(snapshotHash)
	if ok {
		return t, nil
	}
	t, _, err := cli.Cli.AddTorrentSpec(&torrent.TorrentSpec{
		Trackers:    Trackers,
		InfoHash:    snapshotHash,
		DisplayName: snapshotName,
		InfoBytes:   infoBytes,
	})
	return t, err
}

type torrentSpecFromDb struct {
	exists       bool
	snapshotType snapshotsync.SnapshotType
	networkID    uint64
	infoHash     torrent.InfoHash
	infoBytes    []byte
}

func (cli *Client) AddTorrent(ctx context.Context, spec *torrentSpecFromDb) (*torrentSpecFromDb, error) { //nolint: interfacer
	newTorrent := false
	if !spec.exists {
		log.Info("Init new torrent", "snapshot", spec.snapshotType.String())
		newTorrent = true
		var ok bool
		spec.infoHash, ok = TorrentHashes[spec.networkID][spec.snapshotType]
		if !ok {
			return nil, fmt.Errorf("%w type %v, networkID %v", ErrInvalidSnapshot, spec.snapshotType, spec.networkID)
		}
	}
	log.Info("Added torrent spec", "snapshot", spec.snapshotType.String(), "hash", spec.infoHash.String())
	t, err := cli.AddTorrentSpec(spec.snapshotType.String(), spec.infoHash, spec.infoBytes)
	if err != nil {
		return nil, fmt.Errorf("error on add snapshot: %w", err)
	}
	log.Info("Getting infobytes", "snapshot", spec.snapshotType.String())
	spec.infoBytes, err = cli.GetInfoBytes(context.Background(), spec.infoHash)
	if err != nil {
		log.Warn("Init failure", "snapshot", spec.snapshotType.String(), "err", ctx.Err())
		return nil, fmt.Errorf("error on get info bytes: %w", err)
	}
	t.AllowDataDownload()
	t.DownloadAll()
	log.Info("Got infobytes", "snapshot", spec.snapshotType.String(), "file", t.Files()[0].Path())

	if newTorrent {
		return spec, nil
	}
	return nil, nil
}

func (cli *Client) GetInfoBytes(ctx context.Context, snapshotHash metainfo.Hash) ([]byte, error) {
	t, ok := cli.Cli.Torrent(snapshotHash)
	if !ok {
		return nil, errors.New("torrent not added")
	}
	for {
		select {
		case <-ctx.Done():
			return nil, fmt.Errorf("add torrent timeout: %w", ctx.Err())
		case <-t.GotInfo():
			return common.CopyBytes(t.Metainfo().InfoBytes), nil
		default:
			log.Info("Searching infobytes", "seeders", t.Stats().ConnectedSeeders, "active peers", t.Stats().ActivePeers)
			time.Sleep(time.Second * 60)
		}
	}
}

func (cli *Client) Download() {
	log.Info("Start snapshot downloading")
	torrents := cli.Cli.Torrents()
	for i := range torrents {
		t := torrents[i]
		go func(t *torrent.Torrent) {
			defer debug.LogPanic()
			t.AllowDataDownload()
			t.DownloadAll()

			tt := time.Now()
			prev := t.BytesCompleted()
		dwn:
			for {
				if t.Info().TotalLength()-t.BytesCompleted() == 0 {
					log.Info("Dowloaded", "snapshot", t.Name(), "t", time.Since(tt))
					break dwn
				} else {
					stats := t.Stats()
					log.Info("Downloading snapshot",
						"snapshot", t.Name(),
						"%", int(100*(float64(t.BytesCompleted())/float64(t.Info().TotalLength()))),
						"mb", t.BytesCompleted()/1024/1024,
						"diff(kb)", (t.BytesCompleted()-prev)/1024,
						"seeders", stats.ConnectedSeeders,
						"active", stats.ActivePeers,
						"total", stats.TotalPeers)
					prev = t.BytesCompleted()
					time.Sleep(time.Second * 10)

				}

			}
		}(t)
	}
	cli.Cli.WaitAll()

	for _, t := range cli.Cli.Torrents() {
		log.Info("Snapshot seeding", "name", t.Name(), "seeding", t.Seeding())
	}
}

func (cli *Client) GetSnapshots(tx kv.Tx, networkID uint64) (map[snapshotsync.SnapshotType]*snapshotsync.SnapshotsInfo, error) {
	mp := make(map[snapshotsync.SnapshotType]*snapshotsync.SnapshotsInfo)
	networkIDBytes := make([]byte, 8)
	binary.BigEndian.PutUint64(networkIDBytes, networkID)
	err := tx.ForPrefix(kv.SnapshotInfo, append(networkIDBytes, []byte(SnapshotInfoHashPrefix)...), func(k, v []byte) error {
		var hash metainfo.Hash
		if len(v) != metainfo.HashSize {
			return nil
		}
		copy(hash[:], v)
		t, ok := cli.Cli.Torrent(hash)
		if !ok {
			return nil
		}

		var gotInfo bool
		readiness := int32(0)
		select {
		case <-t.GotInfo():
			gotInfo = true
			readiness = int32(100 * (float64(t.BytesCompleted()) / float64(t.Info().TotalLength())))
		default:
		}

		_, tpStr := ParseInfoHashKey(k)
		tp, ok := snapshotsync.SnapshotType_value[tpStr]
		if !ok {
			return fmt.Errorf("incorrect type: %v", tpStr)
		}

		val := &snapshotsync.SnapshotsInfo{
			Type:          snapshotsync.SnapshotType(tp),
			GotInfoByte:   gotInfo,
			Readiness:     readiness,
			SnapshotBlock: SnapshotBlock,
			Dbpath:        filepath.Join(cli.snapshotsDir, t.Files()[0].Path()),
		}
		mp[snapshotsync.SnapshotType(tp)] = val
		return nil
	})
	if err != nil {
		return nil, err
	}

	return mp, nil
}

func (cli *Client) SeedSnapshot(name string, path string) (metainfo.Hash, error) {
	info, err := BuildInfoBytesForSnapshot(path, MdbxFilename)
	if err != nil {
		return [20]byte{}, err
	}

	infoBytes, err := bencode.Marshal(info)
	if err != nil {
		return [20]byte{}, err
	}

	t, err := cli.AddTorrentSpec(name, metainfo.HashBytes(infoBytes), infoBytes)
	if err != nil {
		return [20]byte{}, err
	}
	return t.InfoHash(), nil
}
func (cli *Client) StopSeeding(hash metainfo.Hash) error {
	t, ok := cli.Cli.Torrent(hash)
	if !ok {
		return nil
	}
	ch := t.Closed()
	t.Drop()
	<-ch
	return nil
}

func getTorrentSpec(db kv.Tx, snapshotName snapshotsync.SnapshotType, networkID uint64) (*torrentSpecFromDb, error) {
	snapshotNameS := snapshotName.String()
	var infoHashBytes, infobytes []byte
	var err error
	b := make([]byte, 8)
	binary.BigEndian.PutUint64(b, networkID)
	infoHashBytes, err = db.GetOne(kv.SnapshotInfo, MakeInfoHashKey(snapshotNameS, networkID))
	if err != nil {
		return nil, err
	}
	var infoHash metainfo.Hash
	if infoHashBytes != nil {
		copy(infoHash[:], infoHashBytes[:metainfo.HashSize])
	}

	infobytes, err = db.GetOne(kv.SnapshotInfo, MakeInfoBytesKey(snapshotNameS, networkID))
	if err != nil {
		return nil, err
	}

	return &torrentSpecFromDb{
		exists:       infoHashBytes != nil,
		snapshotType: snapshotName,
		networkID:    networkID,
		infoHash:     infoHash,
		infoBytes:    infobytes,
	}, nil
}

func saveTorrentSpec(db kv.Putter, spec *torrentSpecFromDb) error {
	snapshotNameS := spec.snapshotType.String()
	b := make([]byte, 8)
	binary.BigEndian.PutUint64(b, spec.networkID)
	err := db.Put(kv.SnapshotInfo, MakeInfoHashKey(snapshotNameS, spec.networkID), spec.infoHash.Bytes())
	if err != nil {
		return err
	}
	return db.Put(kv.SnapshotInfo, MakeInfoBytesKey(snapshotNameS, spec.networkID), spec.infoBytes)
}

func MakeInfoHashKey(snapshotName string, networkID uint64) []byte {
	b := make([]byte, 8)
	binary.BigEndian.PutUint64(b, networkID)
	return append(b, []byte(SnapshotInfoHashPrefix+snapshotName)...)
}

func MakeInfoBytesKey(snapshotName string, networkID uint64) []byte {
	b := make([]byte, 8)
	binary.BigEndian.PutUint64(b, networkID)
	return append(b, []byte(SnapshotInfoBytesPrefix+snapshotName)...)
}

// ParseInfoHashKey returns networkID and snapshot name
func ParseInfoHashKey(k []byte) (uint64, string) {
	return binary.BigEndian.Uint64(k), string(bytes.TrimPrefix(k[8:], []byte(SnapshotInfoHashPrefix)))
}

func SnapshotSeeding(chainDB kv.RwDB, cli *Client, name string, snapshotsDir string) error {
	var snapshotBlock uint64
	var hasSnapshotBlock bool
	if err := chainDB.View(context.Background(), func(tx kv.Tx) error {
		v, err := tx.GetOne(kv.BittorrentInfo, kv.CurrentHeadersSnapshotBlock)
		if err != nil && !errors.Is(err, ethdb.ErrKeyNotFound) {
			return err
		}
		hasSnapshotBlock = len(v) == 8
		if hasSnapshotBlock {
			snapshotBlock = binary.BigEndian.Uint64(v)
		} else {
			log.Warn("Snapshot block unknown", "snapshot", name, "v", common.Bytes2Hex(v))
		}
		return nil
	}); err != nil {
		return err
	}

	if hasSnapshotBlock {
		hash, err := cli.SeedSnapshot(name, SnapshotName(snapshotsDir, name, snapshotBlock))
		if err != nil {
			return err
		}
		log.Info("Start seeding", "snapshot", name, "hash", hash.String())
	}
	return nil
}

// verifyHashesIfNeed - in case when file exists, but .torrent doesn't - .torrent file will
// be created, but adding it will require verify file (and pieces) checksum. This may take long time.
// But once it done - result will stored into `piece completion storage` and this func will finish fast (without disk IO).
// Call it after adding all .torrent files to client.
// Don't need call torrent.VerifyData manually
func waitForChecksumVerify(ctx context.Context, torrentClient *torrent.Client) {
	//TODO: tr.VerifyData() - find when to call it
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	go func() {
		interval := time.Second * 5
		logEvery := time.NewTicker(interval)
		defer logEvery.Stop()

		for {
			select {
			case <-ctx.Done():
				break
			case <-logEvery.C:
				var aggBytesCompleted, aggLen int64
				for _, t := range torrentClient.Torrents() {
					aggBytesCompleted += t.BytesCompleted()
					aggLen += t.Length()
				}

				line := fmt.Sprintf(
					"[torrent] verifying snapshots: %s/%s",
					humanize.Bytes(uint64(aggBytesCompleted)),
					humanize.Bytes(uint64(aggLen)),
				)
				log.Info(line)
			}
		}
	}()
	torrentClient.WaitAll() // wait for checksum verify
	cancel()
}

func waitForDownloadAll(ctx context.Context, torrentClient *torrent.Client) {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	go func() {
		interval := time.Second * 5
		logEvery := time.NewTicker(interval)
		defer logEvery.Stop()

		var prevBytesReadUsefulData, aggByteRate int64

		for {
			time.Sleep(interval)
			select {
			case <-ctx.Done():
				break
			case <-logEvery.C:
				var aggBytesCompleted, aggLen int64
				var aggCompletedPieces, aggNumPieces, aggPartialPieces int
				for _, t := range torrentClient.Torrents() {
					var completedPieces, partialPieces int
					psrs := t.PieceStateRuns()
					for _, r := range psrs {
						if r.Complete {
							completedPieces += r.Length
						}
						if r.Partial {
							partialPieces += r.Length
						}
					}
					aggCompletedPieces += completedPieces
					aggPartialPieces += partialPieces
					aggNumPieces = t.NumPieces()

					stats := t.Stats()

					byteRate := int64(time.Second)
					bytesReadUsefulData := stats.BytesReadUsefulData.Int64()
					byteRate *= stats.BytesReadUsefulData.Int64() - prevBytesReadUsefulData
					byteRate /= int64(interval)
					aggByteRate += byteRate

					prevBytesReadUsefulData = bytesReadUsefulData
					aggBytesCompleted += t.BytesCompleted()
					aggLen += t.Length()
				}

				line := fmt.Sprintf(
					"downloading: %s/%s, %d/%d pieces completed (%d partial): %v/s",
					humanize.Bytes(uint64(aggBytesCompleted)),
					humanize.Bytes(uint64(aggLen)),
					aggCompletedPieces,
					aggNumPieces,
					aggPartialPieces,
					humanize.Bytes(uint64(aggByteRate)),
				)
				log.Info(line)
			}
		}
	}()

	torrentClient.WaitAll()
	cancel()
}
