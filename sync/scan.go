package sync

import (
	"bufio"
	"context"
	"io/ioutil"
	"log"
	"os"
	"strings"

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

func LocateFile(fileName string, parentFolder *SyncedFolder, fileType string) (*drive.File, error) {
	comparison := "="
	if fileType == "file" {
		comparison = "!="
	}

	r, err := parentFolder.Drive.Files.List().PageSize(int64(1000)).OrderBy("folder").
		Q("'" + parentFolder.CloudId + "' in parents AND mimeType " + comparison + " 'application/vnd.google-apps.folder' AND name = '" + strings.ReplaceAll(fileName, "'", "\\'") + "' AND trashed = false").
		Fields("nextPageToken, files(*)").
		Do()

	if err != nil {
		return nil, err
	}

	if len(r.Files) == 0 {
		return nil, nil
	}

	return r.Files[0], nil
}

func CreateSyncedFolder(parentSyncedFolder *SyncedFolder, folderName string) *SyncedFolder {
	syncedFolderFullPath := parentSyncedFolder.LocalPath + folderName + "/"

	folder, err := LocateFile(folderName, parentSyncedFolder, "folder")
	if err != nil {
		log.Printf("Search on Drive failed for folder %s. Reason: '%s'", syncedFolderFullPath, err)
		return nil
	}

	if folder == nil {
		log.Printf("Creating folder %s on Drive", syncedFolderFullPath)
		folder = &drive.File{Name: folderName, MimeType: "application/vnd.google-apps.folder", Parents: []string{parentSyncedFolder.CloudId}}
		createdFolder, err := parentSyncedFolder.Drive.Files.Create(folder).Fields("*").Do()
		if err != nil {
			log.Printf("Failed to create folder %s on Drive. Reason: '%s'", syncedFolderFullPath, err)
			return nil
		}
		folder = createdFolder
	}

	return &SyncedFolder{Drive: parentSyncedFolder.Drive, LocalPath: syncedFolderFullPath, CloudId: folder.Id}
}

func CopyFile(parentSyncedFolder *SyncedFolder, file os.FileInfo) {
	fullFilePath := parentSyncedFolder.LocalPath + file.Name()

	cloudFile, err := LocateFile(file.Name(), parentSyncedFolder, "file")
	if err != nil {
		log.Printf("Search on Drive failed for file %s. Reason: '%s'", fullFilePath, err)
		return
	}

	if cloudFile != nil && cloudFile.Size != file.Size() {
		log.Printf("Deleting file %s from Drive", fullFilePath)
		parentSyncedFolder.Drive.Files.Delete(cloudFile.Id).Do()
		cloudFile = nil
	}

	if cloudFile == nil {
		log.Printf("Copying file %s to Drive", fullFilePath)
		contentReader, err := os.Open(fullFilePath)
		if err != nil {
			log.Printf("Error copying file %s: %s", fullFilePath, err)
			return
		}

		cloudFile = &drive.File{Name: file.Name(), Parents: []string{parentSyncedFolder.CloudId}}
		parentSyncedFolder.Drive.Files.Create(cloudFile).Media(bufio.NewReader(contentReader)).Do()
		return
	}

	// log.Printf("Skipping file %s", fullFilePath)
}

func cloudFileIsInFileInfoList(cloudFile *drive.File, files []os.FileInfo) bool {
	for _, f := range files {
		if f.Name() == cloudFile.Name {
			return true
		}
	}

	return false
}

func CompareFolders(folder *SyncedFolder, files []os.FileInfo) {
	// log.Printf("Comparing folder %s to its Cloud counterpart", folder.LocalPath)

	folder.Drive.Files.List().
		Q("'"+folder.CloudId+"' in parents AND trashed = false").Fields("files(id, name)").
		PageSize(int64(1000)).Pages(context.Background(), func(resultSet *drive.FileList) error {
		for _, cloudFile := range resultSet.Files {
			cloudFileExistsLocally := cloudFileIsInFileInfoList(cloudFile, files)
			if !cloudFileExistsLocally {
				log.Printf("File (or Folder) %s was deleted locally. Deleting from Cloud.", folder.LocalPath+cloudFile.Name)
				folder.Drive.Files.Delete(cloudFile.Id).Do()
				continue
			}
		}
		return nil
	})
}

func ScanSyncedFolder(folder *SyncedFolder, foldersToSync chan<- *SyncedFolder) {
	files, err := ioutil.ReadDir(folder.LocalPath)

	if err != nil {
		log.Printf("Error reading folder %s. Skipping.", folder.LocalPath)
		return
	}

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

func StartSync(foldersToSync chan *SyncedFolder) {
	for folder := range foldersToSync {
		if folder != nil {
			ScanSyncedFolder(folder, foldersToSync)
		}
	}
}
