package main

import (
	"fmt"
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
	"github.com/pkg/errors"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
	"go.uber.org/multierr"

	"github.com/danielmmetz/autoplex/pkg/extract"
	"github.com/danielmmetz/autoplex/pkg/finder"
)

var sample = regexp.MustCompile("(?i)sample")

func main() {
	pflag.Duration(
		"frequency",
		1*time.Minute,
		"duration between runs",
	)
	pflag.StringSlice(
		"src",
		[]string{},
		"source directory for downloaded files",
	)
	pflag.StringSlice(
		"dest",
		[]string{},
		"destination directory for extracted files",
	)
	pflag.Parse()
	_ = viper.BindPFlags(pflag.CommandLine)
	frequency := viper.GetDuration("frequency")
	srcs := viper.GetStringSlice("src")
	dests := viper.GetStringSlice("dest")
	if len(srcs) != len(dests) {
		log.Fatal("configuration error: unequal number of sources and destinations")
	}
	pairs := zip(srcs, dests)

	log.Println("running with the following parameters:")
	log.Println("\tfrequency: ", frequency)
	log.Printf("\tpairs: %+v", pairs)

	if err := exec.Command("which", "unrar").Run(); err != nil {
		log.Fatalln("error: could not find unrar")
	}
	tc, err := rpc.New("localhost", "rpcuser", "rpcpass", nil)
	if err != nil {
		log.Fatalln("error intiializing tranmission client: ", err)
	}

	ticker := time.NewTicker(frequency)
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

	if err := work(tc, pairs...); err != nil {
		log.Println(err)
	}

	for range ticker.C {
		select {
		case <-ticker.C:
			fmt.Println("run starting")
			if err := work(tc, pairs...); err != nil {
				log.Println(err)
			}
			fmt.Println("run successful")
		case <-sigs:
			os.Exit(0)
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

func work(tc *rpc.Client, srcDests ...srcDest) error {
	torrents, err := finder.GetFinishedTorrents(tc)
	if err != nil {
		return errors.Wrap(err, "error getting torrents")
	}

	for _, candidate := range torrents {
		var dest string
		for _, p := range srcDests {
			if *candidate.DownloadDir == p.src {
				dest = p.dest
				break
			}
		}
		if dest == "" {
			log.Print("no src-dest pair for processing ", *candidate.DownloadDir)
			continue
		}

		path := filepath.Join(*candidate.DownloadDir, *candidate.Name)
		file, err := os.Open(path)
		if err != nil {
			log.Printf("error opening filepath %s: %v", path, err)
			continue
		}
		stat, err := file.Stat()
		if err != nil {
			log.Printf("error calling Stat() on filepath %s: %v", path, err)
			continue
		}
		if !stat.IsDir() {
			// log.Println("skipping consideration of non-directory:", path)
			continue
		}

		if containsRar, err := processRar(path, dest); err != nil {
			log.Printf("error during processRar(%s): %v", path, err)
			continue
		} else if containsRar { // success. no need to continue trying
			continue
		}
		if _, err := processMKVS(path, dest); err != nil {
			log.Printf("error during processMKVS(%s): %v", path, err)
			continue
		}
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
		log.Println("no .mkv found in", rar.Name())
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

	log.Println("extracting", targetMKVName)
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
	log.Println("successfully extracted rar from", filepath.Base(path))
	return true, nil
}

func processMKVS(path string, destDir string) (containsMKV bool, err error) {
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
		log.Println("linking", filepath.Base(mkvPath))
		if newErr := os.Link(mkvPath, filepath.Join(destDir, filepath.Base(mkvPath))); newErr != nil {
			err = multierr.Append(err, newErr)
			continue
		}
		log.Println("successfully linked", filepath.Base(mkvPath))
	}
	return true, err
}
