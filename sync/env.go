package sync

import (
	"os"
	"runtime"
	"strings"
)

// GetSourceAndDestFolders returns Source, Dest folders to sync
func GetSourceAndDestFolders() (string, string) {
	source := os.Getenv("SOURCE_FOLDER")
	dest := os.Getenv("DEST_FOLDER")

	if source == "" {
		source = "/mnt/sync"

		if runtime.GOOS == "windows" {
			source = "Z:\\"
		}
	}

	if dest == "" {
		dest = "CLOUD"
	}

	if !strings.HasSuffix(source, "/") && !strings.HasSuffix(source, "\\") {
		source = source + "/"
	}

	return strings.TrimSpace(source), strings.TrimSpace(dest)
}
