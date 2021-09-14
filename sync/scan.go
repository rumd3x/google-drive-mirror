package sync

import (
	"bufio"
	"context"
	"io/ioutil"
	"log"
	"os"

	"google.golang.org/api/drive/v3"
)

type SyncedFolder struct {
	Drive     *drive.Service
	LocalPath string
	CloudId   string
}

func EnsureDestFolder(drv *drive.Service, folderName string) *drive.File {
	r, err := drv.Files.List().PageSize(int64(1000)).OrderBy("folder").
		Q("'root' in parents AND mimeType = 'application/vnd.google-apps.folder' AND name = '" + folderName + "' AND trashed = false").
		Fields("nextPageToken, files(*)").
		Do()

	if err != nil {
		log.Fatalf("Fatal error while searching for dest folder: %s", err)
	}

	if len(r.Files) == 0 {
		log.Println("Creating dest folder")

		destFolder := &drive.File{Name: folderName, MimeType: "application/vnd.google-apps.folder"}
		destFolder, err = drv.Files.Create(destFolder).Fields("*").Do()
		if err != nil {
			log.Fatalf("Fatal error while creating dest folder: %s", err)
		}

		log.Println("Dest folder created. ID:", destFolder.Id)
		return destFolder
	}

	log.Println("Dest folder located. ID:", r.Files[0].Id)
	return r.Files[0]
}

func LocateFile(fileName string, parentFolder SyncedFolder, fileType string) *drive.File {
	comparison := "="
	if fileType == "file" {
		comparison = "!="
	}
	r, err := parentFolder.Drive.Files.List().PageSize(int64(1000)).OrderBy("folder").
		Q("'" + parentFolder.CloudId + "' in parents AND mimeType " + comparison + " 'application/vnd.google-apps.folder' AND name = '" + fileName + "' AND trashed = false").
		Fields("nextPageToken, files(*)").
		Do()

	if err != nil {
		return nil
	}

	if len(r.Files) == 0 {
		return nil
	}

	return r.Files[0]
}

func CreateSyncedFolder(parentSyncedFolder SyncedFolder, folderName string) SyncedFolder {
	folder := LocateFile(folderName, parentSyncedFolder, "folder")
	syncedFolderFullPath := parentSyncedFolder.LocalPath + folderName + "/"

	if folder == nil {
		log.Printf("Creating folder %s on Cloud", syncedFolderFullPath)
		folder = &drive.File{Name: folderName, MimeType: "application/vnd.google-apps.folder", Parents: []string{parentSyncedFolder.CloudId}}
		folder, _ = parentSyncedFolder.Drive.Files.Create(folder).Fields("*").Do()
	}

	return SyncedFolder{Drive: parentSyncedFolder.Drive, LocalPath: syncedFolderFullPath, CloudId: folder.Id}
}

func CopyFile(parentSyncedFolder SyncedFolder, file os.FileInfo) {
	fullFilePath := parentSyncedFolder.LocalPath + file.Name()
	cloudFile := LocateFile(file.Name(), parentSyncedFolder, "file")

	if cloudFile != nil && cloudFile.Size != file.Size() {
		log.Printf("Deleting file %s from Cloud", fullFilePath)
		parentSyncedFolder.Drive.Files.Delete(cloudFile.Id).Do()
		cloudFile = nil
	}

	if cloudFile == nil {
		log.Printf("Copying file %s to Cloud", fullFilePath)
		contentReader, err := os.Open(fullFilePath)
		if err != nil {
			log.Printf("Error copying file %s: %s", fullFilePath, err)
			return
		}

		cloudFile = &drive.File{Name: file.Name(), Parents: []string{parentSyncedFolder.CloudId}}
		parentSyncedFolder.Drive.Files.Create(cloudFile).Media(bufio.NewReader(contentReader)).Do()
	}
}

func cloudFileIsInFileInfoList(cloudFile *drive.File, files []os.FileInfo) bool {
	for _, f := range files {
		if f.Name() == cloudFile.Name {
			return true
		}
	}

	return false
}

func CompareFolders(folder SyncedFolder, files []os.FileInfo) {
	folder.Drive.Files.List().Q("'"+folder.CloudId+"' in parents AND trashed = false").Fields("file(id, name)").PageSize(int64(1000)).
		Pages(context.Background(), func(f *drive.FileList) error {
			for _, cloudFile := range f.Files {
				cloudFileExistsLocally := cloudFileIsInFileInfoList(cloudFile, files)
				if !cloudFileExistsLocally {
					log.Printf("File (or Folder) %s was deleted locally. Deleting from Cloud.", folder.LocalPath+cloudFile.Name)
					folder.Drive.Files.Delete(cloudFile.Id).Do()
				}
			}
			return nil
		})
}

func ScanSyncedFolder(folder SyncedFolder, foldersToSync chan<- SyncedFolder) {
	files, _ := ioutil.ReadDir(folder.LocalPath)

	for _, f := range files {

		if f.Name() == "$RECYCLE.BIN" || f.Name() == ".tmp.drivedownload" || f.Name() == "System Volume Information" {
			continue
		}

		if f.IsDir() {
			foldersToSync <- CreateSyncedFolder(folder, f.Name())
			continue
		}

		CopyFile(folder, f)
	}

	CompareFolders(folder, files)
}

func StartSync(foldersToSync chan SyncedFolder) {
	for folder := range foldersToSync {
		ScanSyncedFolder(folder, foldersToSync)
	}
}
