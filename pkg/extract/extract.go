package extract

import (
	"errors"
	"os"
	"strings"
)

// FindRar searches a slice of files for a filename ending with .rar.
// Returns the first such file. Returns a non-nil error if none exists.
func FindRar(files []os.FileInfo) (os.FileInfo, error) {
	for _, file := range files {
		if !file.IsDir() && strings.HasSuffix(file.Name(), ".rar") {
			return file, nil
		}
	}
	return nil, errors.New("no .rar file found")
}

func FindMKV(candidates []string) (string, error) {
	for _, s := range candidates {
		if strings.HasSuffix(s, ".mkv") {
			return s, nil
		}
	}
	return "", errors.New("no .mkv found")
}
