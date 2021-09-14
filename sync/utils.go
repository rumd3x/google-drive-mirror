package sync

import (
	"errors"
	"io/ioutil"
	"os"
)

// Exists returns whether of not given path exists
func Exists(path string) bool {
	_, err := os.Stat(path)
	if err == nil {
		return true
	}

	if errors.Is(err, os.ErrNotExist) {
		return false
	}

	return false
}

// IsDirectory returns whether or not given path is a valid directory
func IsDirectory(path string) bool {
	if !Exists(path) {
		return false
	}

	fileInfo, _ := os.Stat(path)
	return fileInfo.IsDir()
}

// IsFile returns whether or not given path is a valid file
func IsFile(path string) bool {
	if !Exists(path) {
		return false
	}

	return !IsDirectory(path)
}

// FileCount returns the amount of files and folders inside a given path
func FileCount(path string) int {
	if !IsDirectory(path) {
		return 0
	}

	files, err := ioutil.ReadDir(path)
	if err != nil {
		return 0
	}

	return len(files)
}
