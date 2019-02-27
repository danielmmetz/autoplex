package extract

import (
	"os"
	"strings"
)

// FindRar searches a slice of files for a filename ending with .rar.
// Returns the first such file. Returns a nil if none exists.
func FindRar(files []os.FileInfo) os.FileInfo {
	for _, file := range files {
		if !file.IsDir() && strings.HasSuffix(file.Name(), ".rar") {
			return file
		}
	}
	return nil
}

// FindMKV searches a slice of strings for a one ending with .mkv.
// Returns the first such string. Returns nil if none exists.
func FindMKV(candidates []string) string {
	for _, s := range candidates {
		if strings.HasSuffix(s, ".mkv") {
			return s
		}
	}
	return ""
}
