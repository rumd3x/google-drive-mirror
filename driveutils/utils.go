package driveutils

import (
	"context"
	"log"
	"net/http"
	"strings"

	"golang.org/x/oauth2"
	"google.golang.org/api/drive/v3"
	"google.golang.org/api/option"
)

// Retrieve a token, saves the token, then returns the generated client.
func getClient(config *oauth2.Config) *http.Client {
	// The file token.json stores the user's access and refresh tokens, and is
	// created automatically when the authorization flow completes for the first
	// time.
	tokFile := "token.json"
	tok, err := tokenFromFile(tokFile)
	if err != nil {
		tok = getTokenFromWeb(config)
		saveToken(tokFile, tok)
	}
	return config.Client(context.Background(), tok)
}

// Creates a new Google Drive service
func GetDrive() *drive.Service {
	srv, err := drive.NewService(context.Background(), option.WithHTTPClient(getClient(getOAuthConfig())))
	if err != nil {
		log.Fatalf("Unable to retrieve Drive client: %v", err)
	}

	return srv
}

// IsFolder Returns true if provided file is a folder, false otherwise
func IsFolder(file *drive.File) bool {
	if file == nil {
		return false
	}

	return strings.ToLower(file.MimeType) == "application/vnd.google-apps.folder"
}
