package main

import (
	"context"
	"log"
	"runtime"
	"time"

	"google-drive-mirror/driveutils"
	"google-drive-mirror/sync"
)

func main() {
	log.Println("Initializing...")

	drv, tok := driveutils.GetDrive()
	srcPath, dest := sync.GetSourceAndDestFolders()

	if !sync.IsDirectory(srcPath) {
		log.Fatalf("Source Sync Folder '%s' doesn't exist. Exiting.", srcPath)
	}

	log.Printf("Source folder '%s' exists and contains %d files and folders", srcPath, sync.FileCount(srcPath))

	go func() {
		for {
			tokenSrc := driveutils.GetOAuthConfig().TokenSource(context.Background(), tok)
			newTok, err := tokenSrc.Token()

			if err != nil {
				log.Fatal("Failed to renew OAuth Token: ", err)
			}

			if tok.AccessToken != newTok.AccessToken {
				driveutils.SaveToken("token.json", newTok)
				drv, tok = driveutils.GetDrive()
			}

			time.Sleep(30 * time.Minute)
		}
	}()

	time.Sleep(10 * time.Second)
	cloudFolder := sync.EnsureDestFolder(drv, dest)

	rootFolder := sync.SyncedFolder{LocalPath: srcPath, CloudId: cloudFolder.Id, Drive: drv}

	foldersToSync := make(chan *sync.SyncedFolder, 10000000)

	for j := 0; j < runtime.NumCPU(); j++ {
		go sync.StartSync(foldersToSync)
	}

	for {
		log.Printf("Starting Sync Job at Root Folder %s", rootFolder.LocalPath)
		foldersToSync <- &rootFolder
		time.Sleep(time.Hour * 1)
	}
}
