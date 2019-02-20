package main

import (
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	rpc "github.com/hekmon/transmissionrpc"

	"github.com/danielmmetz/autoplex/pkg/extract"
	"github.com/danielmmetz/autoplex/pkg/finder"
)

const destDir = "/media/TV"

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
		files, err := ioutil.ReadDir(path)
		if err != nil {
			log.Printf("error listing files in directory %v: %v", path, err)
			continue
		}
		rar, err := extract.FindRar(files)
		if err != nil {
			// log.Printf("error finding rar in %v: %v", *candidate.Name, err)
			continue
		}

		// List rar contents
		rarPath := filepath.Join(path, rar.Name())
		rawContents, err := exec.Command("unrar", "lbare", rarPath).Output()
		if err != nil {
			log.Printf("error listing contents of %s: %v", rarPath, err)
			continue
		}
		archiveContents := strings.Split(string(rawContents), "\n")

		// Identify the desired file
		targetMKVName, err := extract.FindMKV(archiveContents)
		if err != nil {
			log.Println("no .mkv found in", rar.Name())
			continue
		}

		found, err := finder.Contains(destDir, targetMKVName)
		if err != nil {
			log.Printf("error searching for %s in %s: %v", targetMKVName, destDir, err)
			continue
		} else if found {
			// log.Printf("found %v. skipping extraction", targetMKVName)
			continue
		}
		// Extract to well known path
		f, err := os.Create(filepath.Join(destDir, targetMKVName))
		if err != nil {
			log.Printf("unable to create file %s: %v", filepath.Join(destDir, targetMKVName), err)
			continue
		}

		log.Println("extracting", targetMKVName)
		cmd := exec.Command("unrar", "p", "-inul", rarPath, targetMKVName)
		cmd.Stdout = f
		if err := cmd.Run(); err != nil {
			log.Printf("error while extracting %s: %v", targetMKVName, err)
			_ = os.Remove(f.Name())
			continue
		}
		if err := f.Close(); err != nil {
			log.Printf("error closing %s: %v. removing it", targetMKVName, err)
			_ = os.Remove(f.Name())
			continue
		}
		log.Printf("successfully extracted %s", targetMKVName)
	}
	log.Println("run successful. exiting now...")
}
