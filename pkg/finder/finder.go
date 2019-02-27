package finder

import (
	"os"
	"path/filepath"
	"strings"

	rpc "github.com/hekmon/transmissionrpc"
)

func GetFinishedTorrents(c *rpc.Client) ([]*rpc.Torrent, error) {
	torrents, err := c.TorrentGetAll()
	if err != nil {
		return nil, err
	}
	finished := filterFinished(torrents)
	return finished, nil
}

func filterFinished(in []*rpc.Torrent) []*rpc.Torrent {
	var results []*rpc.Torrent
	for i := range in {
		if in[i].PercentDone != nil && *in[i].PercentDone == 1 {
			results = append(results, in[i])
		}
	}
	return results
}

func Contains(dir, needle string) (bool, error) {
	files := make(map[string]bool)
	err := filepath.Walk(dir, func(_ string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		files[info.Name()] = true
		return nil
	})
	if err != nil {
		return false, err
	}
	return files[needle], nil
}

func FindMKVS(dir string) ([]string, error) {
	var mkvPaths []string
	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		if strings.HasSuffix(info.Name(), ".mkv") {
			mkvPaths = append(mkvPaths, path)
			return nil
		}
		return nil
	})
	return mkvPaths, err
}
