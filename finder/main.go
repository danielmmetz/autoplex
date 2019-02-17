package main

import (
	"log"
	"os"
	"path/filepath"
)

func main() {
	found, err := contains("/Users/dmetz/Downloads", "needle.txt")
	log.Printf("found: %v, err: %v\n", found, err)
}

func contains(dir, needle string) (bool, error) {
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
