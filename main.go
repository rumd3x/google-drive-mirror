package main

import (
	"log"
	"time"

	"google-drive-mirror/driveutils"
	"google-drive-mirror/sync"
)

func main() {
	log.Println("Initializing...")

	drv := driveutils.GetDrive()
	srcPath, dest := sync.GetSourceAndDestFolders()

	if !sync.IsDirectory(srcPath) {
		log.Fatalf("Source Sync Folder '%s' doesn't exist. Exiting.", srcPath)
	}

	log.Printf("Source folder '%s' exists and contains %d files and folders", srcPath, sync.FileCount(srcPath))

	cloudFolder := sync.EnsureDestFolder(drv, dest)

	rootFolder := sync.SyncedFolder{LocalPath: srcPath, CloudId: cloudFolder.Id, Drive: drv}

	foldersToSync := make(chan *sync.SyncedFolder, 10000000)
	for j := 0; j < 20; j++ {
		go sync.StartSync(foldersToSync)
	}

	for {
		log.Printf("Starting Sync Job at Root Folder %s", rootFolder.LocalPath)
		foldersToSync <- &rootFolder
		time.Sleep(time.Hour * 12)
	}
}
