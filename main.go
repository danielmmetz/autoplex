package main

import (
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"

	rpc "github.com/hekmon/transmissionrpc"
	"go.uber.org/multierr"

	"github.com/danielmmetz/autoplex/pkg/extract"
	"github.com/danielmmetz/autoplex/pkg/finder"
)

const destDir = "/media/TV"

var sample = regexp.MustCompile("(?i)sample")

// init evaluates necessary pre-conditions, terminating the program if they fail.
func init() {
	// Ensure unrar exists within $PATH
	if err := exec.Command("which", "unrar").Run(); err != nil {
		log.Fatalln("error: could not find unrar")
	}
}

func main() {
	log.Println("starting...")
	tc, err := rpc.New("localhost", "rpcuser", "rpcpass", nil)
	if err != nil {
		log.Fatalln("error intiializing tranmission client: ", err)
	}

	torrents, err := finder.GetFinishedTorrents(tc)
	if err != nil {
		log.Fatalln("error getting torrents: ", err)
	}

	for _, candidate := range torrents {
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
		if containsRar, err := processRar(path); err != nil {
			log.Printf("error during processRar(%s): %v", path, err)
			continue
		} else if containsRar { // success. no need to continue trying
			continue
		}
		if _, err := processMKVS(path); err != nil {
			log.Printf("error during processMKVS(%s): %v", path, err)
			continue
		}
	}
	log.Println("run successful. exiting now...")
}

// processRar looks in path for .rar files. If present, attempts to find an .mkv
// within the archive and extract it to destDir.
func processRar(path string) (containsRar bool, err error) {
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

	found, err := finder.Contains(destDir, targetMKVName)
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

func processMKVS(path string) (containsMKV bool, err error) {
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
		found, newErr := finder.Contains(destDir, filepath.Base(mkvPath))
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
