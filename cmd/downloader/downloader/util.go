package downloader

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"time"

	"github.com/anacrolix/torrent/bencode"
	"github.com/anacrolix/torrent/metainfo"
	"github.com/ledgerwatch/erigon/cmd/downloader/trackers"
	"github.com/ledgerwatch/erigon/turbo/snapshotsync"
	"github.com/ledgerwatch/erigon/turbo/snapshotsync/snapshothashes"
	"github.com/ledgerwatch/log/v3"
	"github.com/pelletier/go-toml"
)

// DefaultPieceSize - Erigon serves many big files, bigger pieces will reduce
// amount of network announcements, but can't go over 2Mb
// see https://wiki.theory.org/BitTorrentSpecification#Metainfo_File_Structure
const DefaultPieceSize = 2 * 1024 * 1024
const MdbxFilename = "mdbx.dat"

// Trackers - break down by priority tier
var Trackers = [][]string{
	trackers.Best, trackers.Ws, //trackers.Udp, trackers.Https, trackers.Http,
}

func allTorrentFiles(dir string) ([]string, error) {
	files, err := ioutil.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	var res []string
	for _, f := range files {
		if !snapshotsync.IsCorrectFileName(f.Name()) {
			continue
		}
		if f.Size() == 0 {
			continue
		}
		if filepath.Ext(f.Name()) != ".torrent" { // filter out only compressed files
			continue
		}
		res = append(res, f.Name())
	}
	return res, nil
}

func ForEachTorrentFile(root string, walker func(torrentFileName string) error) error {
	files, err := allTorrentFiles(root)
	if err != nil {
		return err
	}
	for _, f := range files {
		torrentFileName := filepath.Join(root, f)
		if _, err := os.Stat(torrentFileName); err != nil {
			if errors.Is(err, os.ErrNotExist) {
				continue
			}
			return err
		}
		if err := walker(torrentFileName); err != nil {
			return err
		}
	}
	return nil
}

func BuildTorrentFilesIfNeed(ctx context.Context, root string) error {
	logEvery := time.NewTicker(20 * time.Second)
	defer logEvery.Stop()

	files, err := allTorrentFiles(root)
	if err != nil {
		return err
	}
	for i, f := range files {
		torrentFileName := path.Join(root, f)
		if _, err := os.Stat(torrentFileName); err != nil {
			if !errors.Is(err, os.ErrNotExist) {
				return err
			}
			info, err := BuildInfoBytesForFile(root, f)
			if err != nil {
				return err
			}
			if err := CreateTorrentFile(root, info); err != nil {
				return err
			}
		}

		select {
		default:
		case <-ctx.Done():
			return ctx.Err()
		case <-logEvery.C:
			log.Info("[torrent] Create .torrent files", "progress", fmt.Sprintf("%d/%d", i, len(files)))
		}
	}
	return nil
}

func BuildInfoBytesForFile(root string, fileName string) (*metainfo.Info, error) {
	info := &metainfo.Info{PieceLength: DefaultPieceSize}
	if err := info.BuildFromFilePath(filepath.Join(root, fileName)); err != nil {
		return nil, err
	}
	return info, nil
}

//nolint
func BuildMetaToml(snapshotsDir string) error {
	//TODO: check existence
	metaFilePath := path.Join(snapshotsDir, "meta.toml")
	if _, err := os.Stat(metaFilePath); err != nil {
		if !errors.Is(err, os.ErrNotExist) {
			return err
		}
		info, err := BuildInfoBytesForFile(snapshotsDir, "meta.toml")
		if err != nil {
			return err
		}
		if err := CreateTorrentFile(snapshotsDir, info); err != nil {
			return err
		}
	}

	metaFile := snapshothashes.Preverified{}
	if err := ForEachTorrentFile(snapshotsDir, func(torrentFilePath string) error {
		mi, err := metainfo.LoadFromFile(torrentFilePath)
		if err != nil {
			return err
		}

		_, fileName := path.Split(torrentFilePath)
		metaFile[fileName] = mi.HashInfoBytes().String()
		return nil
	}); err != nil {
		return err
	}
	b, err := toml.Marshal(metaFile)
	if err != nil {
		return err
	}
	b2, err := json.Marshal(metaFile)
	if err != nil {
		return err
	}
	fmt.Printf("%s\n", b2)
	if err := ioutil.WriteFile(metaFilePath, b, 0600); err != nil {
		return err
	}
	info, err := BuildInfoBytesForFile(snapshotsDir, "meta.toml")
	if err != nil {
		return err
	}
	if err := CreateTorrentFile(snapshotsDir, info); err != nil {
		return err
	}
	//hh, _ := common.HashData(b) // sign?
	//fmt.Printf("%s,%x\n", b, hh)
	return nil
}

func CreateTorrentFileIfNotExists(root string, info *metainfo.Info) error {
	torrentFileName := filepath.Join(root, info.Name+".torrent")
	if _, err := os.Stat(torrentFileName); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return CreateTorrentFile(root, info)
		}
		return err
	}
	return nil
}

func CreateTorrentFile(root string, info *metainfo.Info) error {
	infoBytes, err := bencode.Marshal(info)
	if err != nil {
		return err
	}
	mi := &metainfo.MetaInfo{
		CreationDate: time.Now().Unix(),
		CreatedBy:    "erigon",
		InfoBytes:    infoBytes,
		AnnounceList: Trackers,
	}
	torrentFileName := filepath.Join(root, info.Name+".torrent")

	file, err := os.Create(torrentFileName)
	if err != nil {
		return err
	}
	defer file.Sync()
	defer file.Close()
	if err := mi.Write(file); err != nil {
		return err
	}
	return nil
}
