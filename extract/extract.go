package main

import (
	"errors"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"strings"
)

func main() {
	// Ensure `unrar` exists within $PATH
	if err := exec.Command("which", "unrar").Run(); err != nil {
		log.Fatalln("error: could not find `unrar`")
	}
	// Find the `.rar` file
	files, err := ioutil.ReadDir(".")
	if err != nil {
		log.Fatalln("error listing files in current directory:", err)
	}
	targetRar, err := findRar(files)
	if err != nil {
		log.Fatalln(err)
	}
	// List rar contents
	rawContents, err := exec.Command("unrar", "lbare", targetRar.Name()).Output()
	if err != nil {
		log.Fatalln("error listing archive contents:", err)
	}
	archiveContents := strings.Split(string(rawContents), "\n")

	// Identify the desired file
	targetMKVName, err := findMKV(archiveContents)
	if err != nil {
		log.Fatalln("no media target found")
	}

	// Extract to well known path
	f, err := os.Create(targetMKVName) // TODO include ideal destination path
	if err != nil {
		log.Fatalln("unable to create result file")
	}
	defer f.Close()

	cmd := exec.Command("unrar", "p", targetRar.Name(), targetMKVName)
	cmd.Stdout = f
	if err := cmd.Run(); err != nil {
		log.Fatalln("error while unarchiving:", err)
	}
	log.Printf("successfully extracted %s to %v\n", targetMKVName, f.Name())
}

func findRar(files []os.FileInfo) (os.FileInfo, error) {
	for _, file := range files {
		if !file.IsDir() && strings.HasSuffix(file.Name(), ".rar") {
			return file, nil
		}
	}
	return nil, errors.New("no `.rar` file found")
}

func findMKV(candidates []string) (string, error) {
	for _, s := range candidates {
		if strings.HasSuffix(s, ".mkv") {
			return s, nil
		}
	}
	return "", errors.New("no `.mkv` found")
}
