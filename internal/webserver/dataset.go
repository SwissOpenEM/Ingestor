package webserver

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"path"
	"path/filepath"

	"github.com/SwissOpenEM/Ingestor/internal/core"
	"github.com/SwissOpenEM/Ingestor/internal/datasetaccess"
	"github.com/SwissOpenEM/Ingestor/internal/extglobusservice"
	"github.com/SwissOpenEM/Ingestor/internal/s3upload"
	"github.com/SwissOpenEM/Ingestor/internal/transfertask"
	"github.com/SwissOpenEM/Ingestor/internal/webserver/collections"
	"github.com/SwissOpenEM/Ingestor/internal/webserver/globusauth"
	"github.com/google/uuid"
	"github.com/paulscherrerinstitute/scicat-cli/v3/datasetIngestor"
	"github.com/paulscherrerinstitute/scicat-cli/v3/datasetUtils"
)

func (i *IngestorWebServerImplemenation) DatasetControllerBrowseFilesystem(ctx context.Context, request DatasetControllerBrowseFilesystemRequestObject) (DatasetControllerBrowseFilesystemResponseObject, error) {
	// an internal function used to determine if a folder has subfolders
	folderHasFilesOrSubFolders := func(path string) (bool, bool) {
		folder, err := os.Open(path)
		if err != nil {
			return false, false
		}

		defer folder.Close()
		entries, err := folder.ReadDir(-1) // this could be optimised by doing this in chunks
		if err != nil {
			return false, false
		}
		hasFiles, hasDirs := false, false
		for _, entry := range entries {
			if entry.IsDir() {
				hasDirs = true
			} else {
				hasFiles = true
			}
			if hasDirs && hasFiles {
				break
			}
		}
		return hasFiles, hasDirs
	}

	// if we're at the root, return list of collection locations
	if path.Clean(request.Params.Path) == "/" {
		collections := collections.GetCollectionList(i.pathConfig.CollectionLocations)
		folders := make([]FolderNode, len(collections))

		for c, collection := range collections {
			path := i.pathConfig.CollectionLocations[collection]
			_, hasChildren := folderHasFilesOrSubFolders(path)
			folders[c] = FolderNode{
				Children:        hasChildren,
				Name:            collection,
				Path:            "/" + collection,
				ProbablyDataset: false,
			}
		}

		return DatasetControllerBrowseFilesystem200JSONResponse{
			Folders: folders,
			Total:   uint(len(collections)),
		}, nil
	}

	collectionName, collectionPath, relPath, err := collections.GetPathDetails(i.pathConfig.CollectionLocations, filepath.Clean(request.Params.Path))
	if err != nil {
		return DatasetControllerBrowseFilesystem400TextResponse(err.Error()), nil
	}
	absPath := filepath.Join(collectionPath, relPath)

	// check if path is dir
	err = datasetaccess.IsFolderCheck(absPath)
	if err != nil {
		return DatasetControllerBrowseFilesystem400TextResponse("path is directory check error: " + err.Error()), nil
	}

	// dataset access checks
	if !i.disableAuth {
		err = datasetaccess.CheckUserAccess(ctx, absPath)
		if _, ok := err.(*datasetaccess.AccessError); ok {
			return DatasetControllerBrowseFilesystem401TextResponse("unauthorized: " + err.Error()), nil
		} else if err != nil {
			slog.Error("user access error", "error", err.Error())
			return DatasetControllerBrowseFilesystem500TextResponse("internal server error: user access error"), nil
		}
	}

	// get page values
	page := uint(1)
	pageSize := uint(10)
	if request.Params.Page != nil {
		page = max(*request.Params.Page, 1)
	}
	if request.Params.PageSize != nil {
		pageSize = min(*request.Params.PageSize, 100)
	}

	start := (page - 1) * pageSize
	end := page * pageSize
	folderCounter := uint(0)
	folders := make([]FolderNode, pageSize)

	// flat directory walk to put a section of folders into the 'folders' slice
	err = filepath.WalkDir(absPath, func(currPath string, d os.DirEntry, err error) error {
		if err != nil {
			return nil // skip errored elements
		}
		if d.IsDir() && currPath != absPath {
			if folderCounter >= start && folderCounter < end {
				_, hasChildren := folderHasFilesOrSubFolders(currPath)
				relativePath, _ := filepath.Rel(collectionPath, currPath)

				folders[folderCounter-start].Name = d.Name()
				folders[folderCounter-start].Path = "/" + collectionName + "/" + filepath.ToSlash(relativePath)
				folders[folderCounter-start].Children = hasChildren
				folders[folderCounter-start].ProbablyDataset = datasetaccess.IsDatasetFolder(currPath)
			}
			folderCounter++
			return filepath.SkipDir // prevent recursing into subfolders
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	if folderCounter >= start {
		folders = folders[0 : min(end, folderCounter)-start]
	} else {
		folders = []FolderNode{}
	}

	return DatasetControllerBrowseFilesystem200JSONResponse{
		Folders: folders,
		Total:   uint(folderCounter),
	}, nil
}

func (i *IngestorWebServerImplemenation) DatasetControllerIngestDataset(ctx context.Context, request DatasetControllerIngestDatasetRequestObject) (DatasetControllerIngestDatasetResponseObject, error) {
	// get sourcefolder from metadata
	var metadata map[string]interface{}
	err := json.Unmarshal([]byte(request.Body.MetaData), &metadata)
	if err != nil {
		return DatasetControllerIngestDataset400TextResponse(err.Error()), nil
	}

	sourceFolder, ok := metadata["sourceFolder"].(string)
	if !ok {
		return DatasetControllerIngestDataset400TextResponse("sourceFolder is not present in the metadata"), nil
	}
	cleanedSourceFolder := filepath.Clean(sourceFolder)

	// get required user info from metadata
	ownerGroup, ok := metadata["ownerGroup"].(string)
	if !ok {
		return DatasetControllerIngestDataset400TextResponse(fmt.Sprintf("Missing key %s in metadata", "ownerGroup")), nil
	}

	contactEmail, ok := metadata["contactEmail"].(string)
	if !ok {
		return DatasetControllerIngestDataset400TextResponse(fmt.Sprintf("Missing key %s in metadata", "contactEmail")), nil
	}

	ownerUser, ok := metadata["owner"].(string)
	if !ok {
		return DatasetControllerIngestDataset400TextResponse(fmt.Sprintf("Missing key %s in metadata", "owner")), nil
	}

	// the sourceFolder attribute's first folder indicates the collection location 'key' (should be the collection's base directory name)
	folderPath, err := collections.GetDatasetAbsolutePath(i.pathConfig.CollectionLocations, cleanedSourceFolder)
	if err != nil {
		return DatasetControllerIngestDataset400TextResponse(err.Error()), nil
	}

	// the convention for the sourceFolder in Scicat is to have the full path where the dataset was collected from
	metadata["sourceFolder"] = folderPath

	// check if folder exists
	err = datasetaccess.IsFolderCheck(folderPath)
	if err != nil {
		return DatasetControllerIngestDataset400TextResponse(fmt.Sprintf("dataset location lookup error: %s", err.Error())), nil
	}

	// dataset access checks
	if !i.disableAuth {
		err = datasetaccess.CheckUserAccess(ctx, folderPath)
		if _, ok := err.(*datasetaccess.AccessError); ok {
			return DatasetControllerIngestDataset401TextResponse("unauthorized: " + err.Error()), nil
		} else if err != nil {
			slog.Error("user access error", "error", err.Error())
			return DatasetControllerIngestDataset500TextResponse("internal server error: user access error"), nil
		}
	}

	// do catalogue insertion
	isOnCentralDisk := i.taskQueue.GetTransferMethod() == transfertask.TransferNone
	datasetID, _, fileList, username, err := core.AddDatasetToScicat(metadata, folderPath, i.taskQueue.Config.Transfer.StorageLocation, request.Body.UserToken, i.taskQueue.Config.Scicat.Host, isOnCentralDisk)
	if err != nil {
		return DatasetControllerIngestDataset400TextResponse(err.Error()), nil
	}

	// set auto-archival parameter
	autoArchive := true
	if request.Body.AutoArchive != nil {
		autoArchive = *request.Body.AutoArchive
	}

	// add transfer job
	var taskID uuid.UUID
	switch i.taskQueue.GetTransferMethod() {
	case transfertask.TransferGlobus:
		taskID, err = i.addGlobusTransferTask(ctx, datasetID, fileList, folderPath, username, ownerUser, ownerGroup, autoArchive, contactEmail)
	case transfertask.TransferExtGlobus:
		jobID, err := i.addExtGlobusTransferTask(ctx, datasetID, fileList, autoArchive, request.Body.UserToken)
		if err != nil {
			if reqErr, ok := err.(*extglobusservice.RequestError); ok {
				if reqErr.Code() < 500 {
					return DatasetControllerIngestDataset400TextResponse(fmt.Sprintf("Transfer request server refused with Code: '%d', Message: '%s', Details: '%s'", reqErr.Code(), reqErr.Error(), reqErr.Details())), nil
				}
			}
			return DatasetControllerIngestDataset400TextResponse(fmt.Sprintf("Transfer request - unknown error: %s", err.Error())), nil
		}
		return DatasetControllerIngestDataset200JSONResponse{
			DatasetId:  datasetID,
			TransferId: getPointerOrNil(jobID),
			Status:     getStrPointerOrNil("started"),
		}, nil
	case transfertask.TransferS3:
		taskID, err = i.addS3TransferTask(ctx, datasetID, fileList, folderPath, ownerUser, ownerGroup, autoArchive, contactEmail, request.Body.UserToken)
	case transfertask.TransferNone:
		if autoArchive {
			user, _, err := datasetUtils.GetUserInfoFromToken(http.DefaultClient, i.taskQueue.Config.Scicat.Host, request.Body.UserToken)
			if err != nil {
				return nil, err
			}

			copies := 1
			_, err = datasetUtils.CreateArchivalJob(http.DefaultClient, i.taskQueue.Config.Scicat.Host, user, ownerGroup, []string{datasetID}, &copies, nil)
			if err != nil {
				return nil, err
			}
		}

		// return response
		return DatasetControllerIngestDataset200JSONResponse{
			DatasetId: datasetID,
			Status:    getStrPointerOrNil("finished"),
		}, nil
	}
	if err != nil {
		if _, ok := err.(*os.PathError); ok {
			return nil, fmt.Errorf("could not create the task due to a path error: %s", err.Error())
		} else {
			return DatasetControllerIngestDataset400TextResponse(fmt.Sprintf("You don't have permissions to access the dataset folder or it doesn't exist: %s", err.Error())), nil
		}
	}

	// schedule transfer job
	err = i.taskQueue.ScheduleTask(taskID)
	if err != nil {
		return DatasetControllerIngestDataset400TextResponse(fmt.Sprintf("error when scheduling task: %s", err.Error())), nil
	}

	status := "started"
	idString := taskID.String()
	return DatasetControllerIngestDataset200JSONResponse{
		DatasetId:  datasetID,
		TransferId: getPointerOrNil(idString),
		Status:     &status,
	}, nil
}

func (i *IngestorWebServerImplemenation) addGlobusTransferTask(ctx context.Context, datasetID string, fileList []datasetIngestor.Datafile, folderPath string, username string, ownerUser string, ownerGroup string, autoArchive bool, contactEmail string) (uuid.UUID, error) {
	taskID := uuid.New()
	transferObjects := map[string]interface{}{}

	client, err := globusauth.GetClientFromSession(ctx, i.globusAuthConf, i.sessionDuration, i.secureCookies)
	if err != nil {
		return uuid.UUID{}, err
	}

	// |-> globus dependencies
	// add transfer dependencies to the transferObjects map
	transferObjects["globus_client"] = client
	transferObjects["username"] = username

	err = i.taskQueue.AddTransferTask(datasetID, fileList, taskID, folderPath, ownerUser, ownerGroup, contactEmail, autoArchive, transferObjects)
	if err != nil {
		return uuid.UUID{}, err
	}
	return taskID, nil
}

func (i *IngestorWebServerImplemenation) addExtGlobusTransferTask(ctx context.Context, datasetID string, fileList []datasetIngestor.Datafile, autoArchive bool, scicatToken string) (string, error) {
	filesToTransfer := make([]extglobusservice.FileToTransfer, len(fileList))
	for i, file := range fileList {
		filesToTransfer[i].Path = file.Path
		filesToTransfer[i].IsSymlink = file.IsSymlink
	}
	return extglobusservice.RequestExternalTransferTask(
		ctx,
		i.taskQueue.Config.Transfer.ExtGlobus.TransferServiceURL,
		scicatToken,
		i.taskQueue.Config.Transfer.ExtGlobus.SrcFacility,
		i.taskQueue.Config.Transfer.ExtGlobus.DstFacility,
		datasetID,
		&filesToTransfer,
	)
}

func (i *IngestorWebServerImplemenation) addS3TransferTask(ctx context.Context, datasetID string, fileList []datasetIngestor.Datafile, folderPath string, ownerUser string, ownerGroup string, autoArchive bool, contactEmail string, scicatToken string) (uuid.UUID, error) {
	taskID := uuid.New()
	transferObjects := map[string]interface{}{}

	// access and refresh token need be fetched at this point from the archiver backend since user token could expire
	accessToken, refreshToken, expiresIn, err := s3upload.GetTokens(ctx, i.taskQueue.Config.Transfer.S3.Endpoint, scicatToken)
	if err != nil {
		return uuid.UUID{}, err
	}
	transferObjects["accessToken"] = accessToken
	transferObjects["refreshToken"] = refreshToken
	transferObjects["expires_in"] = expiresIn

	filteredFileList := []datasetIngestor.Datafile{}
	for _, f := range fileList {
		info, _ := os.Stat(path.Join(folderPath, f.Path))
		if info.IsDir() {
			continue
		}
		filteredFileList = append(filteredFileList, f)
	}
	err = i.taskQueue.AddTransferTask(datasetID, filteredFileList, taskID, folderPath, ownerUser, ownerGroup, contactEmail, autoArchive, transferObjects)
	if err != nil {
		return uuid.UUID{}, err
	}
	return taskID, nil
}

func safeSubslice[T any](s []T, start, end uint) []T {
	sLen := uint(len(s))
	if start >= sLen {
		return []T{}
	}
	if end > sLen {
		end = sLen
	}
	return s[start:end]
}
