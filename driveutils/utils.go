package driveutils

import (
	"context"
	"log"
	"strings"

	"golang.org/x/oauth2"
	"google.golang.org/api/drive/v3"
	"google.golang.org/api/option"
)

// Retrieve a token, saves the token, and returns it.
func getToken(config *oauth2.Config) *oauth2.Token {
	// The file token.json stores the user's access and refresh tokens, and is
	// created automatically when the authorization flow completes for the first
	// time.
	tokFile := "token.json"
	tok, err := tokenFromFile(tokFile)
	if err != nil {
		tok = getTokenFromWeb(config)
		SaveToken(tokFile, tok)
		log.Println("OAuth token renewed.")
	}
	return tok
}

// Creates a new Google Drive service
func GetDrive() (*drive.Service, *oauth2.Token) {
	token := getToken(GetOAuthConfig())
	srv, err := drive.NewService(context.Background(), option.WithHTTPClient(GetOAuthConfig().Client(context.Background(), token)))
	if err != nil {
		log.Fatalf("Unable to retrieve Drive client: %v", err)
	}

	return srv, token
}

// IsFolder Returns true if provided file is a folder, false otherwise
func IsFolder(file *drive.File) bool {
	if file == nil {
		return false
	}

	return strings.ToLower(file.MimeType) == "application/vnd.google-apps.folder"
}
