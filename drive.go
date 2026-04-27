package gcputil

import (
	"context"
	"fmt"
	"io"
	"os"

	"golang.org/x/oauth2/google"
	"google.golang.org/api/drive/v3"
	"google.golang.org/api/option"
)

func CreateDriveService(credentialsFilename string, tokenFilename string) (*drive.Service, context.Context, error) {
	b, err := os.ReadFile(credentialsFilename)
	if err != nil {
		return nil, nil, fmt.Errorf("error reading credentials file: %v", err)
	}

	ctx := context.Background()

	// Define the necessary scopes (Read/Write)
	credentialsConfig, err := google.ConfigFromJSON(b, drive.DriveScope)
	if err != nil {
		return nil, nil, fmt.Errorf("error get credentials config from json: %v", err)
	}
	client, err := GetClientFromTokenOrWeb(tokenFilename, credentialsConfig)
	if err != nil {
		return nil, nil, fmt.Errorf("error get client from token or web: %v", err)
	}

	srv, err := drive.NewService(ctx, option.WithHTTPClient(client))
	if err != nil {
		return srv, ctx, fmt.Errorf("unable to retrieve drive service: %v", err)
	}

	return srv, ctx, nil
}

func ListFilesInFolder(srv *drive.Service, folderId string) ([]*drive.File, error) {
	// Construct the query to list children of the specific folder ID
	query := fmt.Sprintf("'%s' in parents and trashed = false", folderId)

	r, err := srv.Files.List().
		Q(query).
		Fields("files(id, name, mimeType)").
		Do()
	if err != nil {
		return nil, fmt.Errorf("error listing file: %v", err)
	}

	// for _, file := range r.Files {
	// 	fmt.Printf("%s (%s) [%s]\n", file.Name, file.Id, file.MimeType)
	// }
	return r.Files, nil
}

func GetFolderIdByNameAndParent(srv *drive.Service, folderName string, parentId string) (*drive.File, error) {
	query := fmt.Sprintf("name = '%s' and '%s' in parents and mimeType = 'application/vnd.google-apps.folder' and trashed = false", folderName, parentId)

	// fmt.Printf("query: %s\n", query)

	r, err := srv.Files.List().
		Q(query).
		Fields("files(id, name)").
		PageSize(1).
		Do()
	if err != nil {
		return nil, fmt.Errorf("unable to retrieve files: %v", err)
	}

	if len(r.Files) == 0 {
		return nil, fmt.Errorf("no folder found with name %s in parent %s", folderName, parentId)
	}
	return r.Files[0], nil
}

func ListFilesInPaths(srv *drive.Service, paths []string) ([]*drive.File, *drive.File, error) {
	walkFolderId := "root"
	var walkFolder *drive.File = nil
	for pathIdx := range paths {
		nextFile, err := GetFolderIdByNameAndParent(srv, paths[pathIdx], walkFolderId)
		if err != nil {
			return nil, nil, fmt.Errorf("error get folder id by name: %v", err)
		}
		walkFolder = nextFile
		walkFolderId = nextFile.Id
	}
	files, err := ListFilesInFolder(srv, walkFolderId)
	if err != nil {
		return files, walkFolder, fmt.Errorf("error list file in folder: %v", err)
	}
	return files, walkFolder, nil
}

func DownloadBinaryFile(srv *drive.Service, fileId string, destPath string) error {
	// 1. Create the download request
	// The library automatically handles the 'alt=media' parameter
	resp, err := srv.Files.Get(fileId).Download()
	if err != nil {
		return fmt.Errorf("unable to download file: %v", err)
	}
	defer resp.Body.Close()

	// 2. Save the content to a local file
	out, err := os.Create(destPath)
	if err != nil {
		return fmt.Errorf("error create local file: %v", err)
	}
	defer out.Close()

	_, err = io.Copy(out, resp.Body)
	if err != nil {
		return fmt.Errorf("error writing to local file: %v", err)
	}
	return nil
}

func UploadOrReplace(srv *drive.Service, filename string, parentId string, localFilename string) (*drive.File, error) {
	f, err := os.Open(localFilename)
	if err != nil {
		return nil, fmt.Errorf("error opening file: %v", err)
	}
	defer f.Close()

	// 1. Search for existing file with same name in the specific folder
	query := fmt.Sprintf("name = '%s' and '%s' in parents and trashed = false", filename, parentId)
	r, err := srv.Files.List().Q(query).Fields("files(id)").Do()
	if err != nil {
		return nil, fmt.Errorf("error opening file: %v", err)
	}

	if len(r.Files) > 0 {
		// 2. Found! Update the existing file
		existingFileId := r.Files[0].Id
		file, err := srv.Files.Update(existingFileId, nil).Media(f).Do()
		if err != nil {
			return file, fmt.Errorf("error updating file: %v", err)
		}
		return file, nil
	}

	// 3. Not found. Create a new file
	newFileMetadata := &drive.File{
		Name:    filename,
		Parents: []string{parentId},
	}
	file, err := srv.Files.Create(newFileMetadata).Media(f).Do()
	if err != nil {
		return file, fmt.Errorf("error creating file: %v", err)
	}
	return file, nil
}
