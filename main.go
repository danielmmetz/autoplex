package main

import (
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"regexp"
	"strings"
	"syscall"
	"time"

	rpc "github.com/hekmon/transmissionrpc"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
	"go.uber.org/multierr"

	"github.com/danielmmetz/autoplex/pkg/extract"
	"github.com/danielmmetz/autoplex/pkg/finder"
)

var sample = regexp.MustCompile("(?i)sample")

func main() {
	host := pflag.String("host", "localhost", "host at which to access transmission")
	frequency := pflag.Duration("frequency", 1*time.Minute, "duration between runs")
	srcs := pflag.StringSlice("src", []string{}, "source directory for downloaded files")
	dests := pflag.StringSlice("dest", []string{}, "destination directory for extracted files")
	modeF := pflag.String("mode", "link", "method by which to create the destination files (copy or link)")
	pflag.Parse()
	_ = viper.BindPFlags(pflag.CommandLine)
	var m mode
	switch *modeF {
	case "link":
		m = link
	case "copy":
		m = copy
	default:
		log.Fatal("configuration error: mode must be either link or copy")
	}

	if len(*srcs) != len(*dests) {
		log.Fatal("configuration error: unequal number of sources and destinations")
	}
	pairs := zip(*srcs, *dests)

	log.Print("running with the following parameters:")
	log.Print("\tmode: ", m)
	log.Print("\tfrequency: ", frequency)
	log.Printf("\tpairs: %+v", pairs)

	if err := exec.Command("which", "unrar").Run(); err != nil {
		log.Fatalln("error: could not find unrar")
	}
	tc, err := rpc.New(*host, "rpcuser", "rpcpass", nil)
	if err != nil {
		log.Fatalln("error intiializing tranmission client: ", err)
	}

	ticker := time.NewTicker(*frequency)
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

	w := worker{mode: m, tc: tc, srcDestPairs: pairs, retries: 1}
	if err := w.work(); err != nil {
		log.Print(err)
	}

	for {
		select {
		case <-ticker.C:
			log.Print("run starting")
			if err := w.work(); err != nil {
				log.Print(err)
			}
			log.Print("run successful")
		case <-sigs:
			log.Print("exiting")
			return
		}
	}

}

type srcDest struct {
	src  string
	dest string
}

func zip(srcs, dests []string) []srcDest {
	pairs := []srcDest{}
	for i := range srcs {
		if i >= len(dests) {
			break
		}
		pairs = append(pairs, srcDest{src: srcs[i], dest: dests[i]})
	}
	return pairs
}

type worker struct {
	mode         mode
	tc           *rpc.Client
	srcDestPairs []srcDest
	retries      int
	resultCache  resultCache
}

type mode string

const copy mode = "copy"
const link mode = "link"

type resultCache struct {
	success  map[string]bool
	attempts map[string]int
}

func (c *resultCache) RecordAttempt(key string, success bool) {
	if c.success == nil {
		c.success = make(map[string]bool)
	}
	if c.attempts == nil {
		c.attempts = make(map[string]int)
	}

	c.success[key] = success
	c.attempts[key] = c.attempts[key] + 1
}

func (w *worker) work() error {
	torrents, err := finder.GetFinishedTorrents(w.tc)
	if err != nil {
		return fmt.Errorf("error getting torrents: %w", err)
	}

	for _, candidate := range torrents {
		if w.resultCache.success[*candidate.Name] || w.resultCache.attempts[*candidate.Name] >= w.retries {
			continue
		}

		var dest string
		for _, p := range w.srcDestPairs {
			if *candidate.DownloadDir == p.src {
				dest = p.dest
				break
			}
		}
		if dest == "" {
			log.Print("no src-dest pair for processing ", *candidate.DownloadDir)
			w.resultCache.RecordAttempt(*candidate.Name, false)
			continue
		}

		path := filepath.Join(*candidate.DownloadDir, *candidate.Name)
		file, err := os.Open(path)
		if err != nil {
			log.Printf("error opening filepath %s: %v", path, err)
			w.resultCache.RecordAttempt(*candidate.Name, false)
			continue
		}
		stat, err := file.Stat()
		if err != nil {
			log.Printf("error calling Stat() on filepath %s: %v", path, err)
			w.resultCache.RecordAttempt(*candidate.Name, false)
			continue
		}
		if !stat.IsDir() {
			w.resultCache.RecordAttempt(*candidate.Name, false)
			continue
		}

		if containsRar, err := processRar(path, dest); err != nil {
			log.Printf("error during processRar(%s): %v", path, err)
			w.resultCache.RecordAttempt(*candidate.Name, false)
			continue
		} else if containsRar { // success. no need to continue trying
			w.resultCache.RecordAttempt(*candidate.Name, true)
			continue
		}
		containsMKV, err := w.processMKVS(path, dest)
		if err != nil {
			w.resultCache.RecordAttempt(*candidate.Name, false)
			log.Printf("error during processMKVS(%s): %v", path, err)
			continue
		}
		w.resultCache.RecordAttempt(*candidate.Name, containsMKV)
	}
	return nil
}

// processRar looks in path for .rar files. If present, attempts to find an .mkv
// within the archive and extract it to destDir.
func processRar(path string, destDir string) (containsRar bool, err error) {
	files, err := ioutil.ReadDir(path)
	if err != nil {
		log.Printf("error listing files in directory %v: %v", path, err)
		return false, err
	}
	rar := extract.FindRar(files)
	if rar == nil {
		// log.Printf("error finding rar in %v: %v", *candidate.Name, err)
		return false, err
	}

	// List rar contents
	rarPath := filepath.Join(path, rar.Name())
	rawContents, err := exec.Command("unrar", "lbare", rarPath).Output()
	if err != nil {
		log.Printf("error listing contents of %s: %v", rarPath, err)
		return true, err
	}
	archiveContents := strings.Split(string(rawContents), "\n")

	// Identify the desired file
	targetMKVName := extract.FindMKV(archiveContents)
	if targetMKVName == "" {
		log.Print("no .mkv found in ", rar.Name())
		return true, err
	}

	found, err := finder.Contains(targetMKVName, destDir)
	if err != nil {
		log.Printf("error searching for %s in %s: %v", targetMKVName, destDir, err)
		return true, err
	} else if found {
		// log.Printf("found %v. skipping extraction", targetMKVName)
		return true, err
	}
	// Extract to well known path
	f, err := os.Create(filepath.Join(destDir, targetMKVName))
	if err != nil {
		log.Printf("unable to create file %s: %v", filepath.Join(destDir, targetMKVName), err)
		return true, err
	}

	log.Print("extracting ", targetMKVName)
	cmd := exec.Command("unrar", "p", "-inul", rarPath, targetMKVName)
	cmd.Stdout = f
	if err := cmd.Run(); err != nil {
		log.Printf("error while extracting %s: %v", targetMKVName, err)
		_ = os.Remove(f.Name())
		return true, err
	}
	if err := f.Close(); err != nil {
		log.Printf("error closing %s: %v. removing it", targetMKVName, err)
		_ = os.Remove(f.Name())
		return true, err
	}
	log.Print("successfully extracted rar from ", filepath.Base(path))
	return true, nil
}

func (w *worker) processMKVS(path string, destDir string) (containsMKV bool, err error) {
	mkvPaths, err := finder.FindMKVS(path)
	if err != nil {
		log.Printf("error finding mkv in directory %v: %v", path, err)
		return false, err
	} else if len(mkvPaths) == 0 {
		return false, nil
	}
	for _, mkvPath := range mkvPaths {
		if sample.MatchString(mkvPath) {
			continue
		}
		found, newErr := finder.Contains(filepath.Base(mkvPath), destDir)
		if newErr != nil {
			log.Printf("error searching for %s in %s: %v", filepath.Base(mkvPath), destDir, err)
			err = multierr.Append(err, newErr)
			continue
		} else if found {
			// log.Printf("found %v. skipping linking", filepath.Base(mkvPath))
			continue
		}
		switch w.mode {
		case link:
			log.Print("linking ", filepath.Base(mkvPath))
			if newErr := os.Link(mkvPath, filepath.Join(destDir, filepath.Base(mkvPath))); newErr != nil {
				err = multierr.Append(err, newErr)
				continue
			}
			log.Print("successfully linked ", filepath.Base(mkvPath))
		case copy:
			log.Print("copying ", filepath.Base(mkvPath))
			if newErr := copyMKV(mkvPath, destDir); newErr != nil {
				err = multierr.Append(err, newErr)
				continue
			}
			log.Print("successfully copied ", filepath.Base(mkvPath))
		}
	}
	return true, err
}

func copyMKV(mkvPath, destDir string) error {
	original, err := os.Open(mkvPath)
	if err != nil {
		return fmt.Errorf("opening %s: %w", mkvPath, err)
	}
	defer original.Close()

	f, err := os.Create(filepath.Join(destDir, filepath.Base(mkvPath)))
	if err != nil {
		return fmt.Errorf("creating destination file: %w", err)
	}
	defer f.Close()

	if _, err := io.Copy(f, original); err != nil {
		return fmt.Errorf("copying %s: %w", filepath.Base(mkvPath), err)
	}
	return nil
}
