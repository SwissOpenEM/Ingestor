package webserver

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/SwissOpenEM/Ingestor/internal/core"
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
	folder, err := os.Stat(absPath)
	if err != nil {
		if os.IsNotExist(err) {
			return DatasetControllerBrowseFilesystem400TextResponse("path does not exist or is invalid"), nil
		} else {
			return nil, err
		}
	}
	if !folder.IsDir() {
		return DatasetControllerBrowseFilesystem400TextResponse("path does not point to a folder"), nil
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
				hasFiles, hasChildren := folderHasFilesOrSubFolders(currPath)
				relativePath, _ := filepath.Rel(collectionPath, currPath)

				folders[folderCounter-start].Name = d.Name()
				folders[folderCounter-start].Path = "/" + collectionName + "/" + filepath.ToSlash(relativePath)
				folders[folderCounter-start].Children = hasChildren
				folders[folderCounter-start].ProbablyDataset = hasFiles
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

	// adapt source folder attribute to Globus collection path **only if** using external Globus transfer request service
	//   as the service uses it to find the dataset's folder (due to security concerns)
	if i.taskQueue.GetTransferMethod() == transfertask.TransferExtGlobus {
		metadata["sourceFolder"] = strings.TrimPrefix(folderPath, i.taskQueue.Config.Transfer.ExtGlobus.CollectionRootPath)
	}

	// check if folder exists
	err = core.CheckIfFolderExists(folderPath)
	if err != nil {
		return DatasetControllerIngestDataset400TextResponse(fmt.Sprintf("dataset location lookup error: %s", err.Error())), nil
	}

	// do catalogue insertion
	datasetId, _, fileList, username, err := core.AddDatasetToScicat(metadata, folderPath, i.taskQueue.Config.Transfer.StorageLocation, request.Body.UserToken, i.taskQueue.Config.Scicat.Host)
	if err != nil {
		return DatasetControllerIngestDataset400TextResponse(err.Error()), nil
	}

	// set auto-archival parameter
	autoArchive := true
	if request.Body.AutoArchive != nil {
		autoArchive = *request.Body.AutoArchive
	}

	// add transfer job
	var taskId uuid.UUID
	switch i.taskQueue.GetTransferMethod() {
	case transfertask.TransferGlobus:
		taskId, err = i.addGlobusTransferTask(ctx, datasetId, fileList, folderPath, username, ownerUser, ownerGroup, autoArchive, contactEmail)
	case transfertask.TransferExtGlobus:
		jobId, err := i.addExtGlobusTransferTask(ctx, datasetId, fileList, autoArchive, request.Body.UserToken)
		if err != nil {
			if reqErr, ok := err.(*extglobusservice.RequestError); ok {
				if reqErr.Code() < 500 {
					return DatasetControllerIngestDataset400TextResponse(fmt.Sprintf("Transfer request server refused with Code: '%d', Message: '%s', Details: '%s'", reqErr.Code(), reqErr.Error(), reqErr.Details())), nil
				}
			}
			return DatasetControllerIngestDataset400TextResponse(fmt.Sprintf("Transfer request - unknown error: %s", err.Error())), nil
		}
		return DatasetControllerIngestDataset200JSONResponse{
			TransferId: jobId,
			Status:     getStrPointerOrNil("started"),
		}, nil
	case transfertask.TransferS3:
		taskId, err = i.addS3TransferTask(ctx, datasetId, fileList, folderPath, ownerUser, ownerGroup, autoArchive, contactEmail, request.Body.UserToken)
	case transfertask.TransferNone:
		// mark dataset as archivable
		user, _, err := datasetUtils.GetUserInfoFromToken(http.DefaultClient, i.taskQueue.Config.Scicat.Host, request.Body.UserToken)
		if err != nil {
			return nil, err
		}
		err = datasetIngestor.MarkFilesReady(http.DefaultClient, i.taskQueue.Config.Scicat.Host, datasetId, user)
		if err != nil {
			return nil, err
		}

		// auto archive
		if autoArchive {
			copies := 1
			_, err = datasetUtils.CreateArchivalJob(http.DefaultClient, i.taskQueue.Config.Scicat.Host, user, ownerGroup, []string{datasetId}, &copies)
		}

		// return response
		return DatasetControllerIngestDataset200JSONResponse{
			TransferId: "no-transfer",
			Status:     getStrPointerOrNil("finished"),
		}, err
	}
	if err != nil {
		if _, ok := err.(*os.PathError); ok {
			return nil, fmt.Errorf("could not create the task due to a path error: %s", err.Error())
		} else {
			return DatasetControllerIngestDataset400TextResponse(fmt.Sprintf("You don't have permissions to access the dataset folder or it doesn't exist: %s", err.Error())), nil
		}
	}

	// schedule transfer job
	err = i.taskQueue.ScheduleTask(taskId)
	if err != nil {
		return DatasetControllerIngestDataset400TextResponse(fmt.Sprintf("error when scheduling task: %s", err.Error())), nil
	}

	status := "started"
	idString := taskId.String()
	return DatasetControllerIngestDataset200JSONResponse{
		TransferId: idString,
		Status:     &status,
	}, nil
}

func (i *IngestorWebServerImplemenation) addGlobusTransferTask(ctx context.Context, datasetId string, fileList []datasetIngestor.Datafile, folderPath string, username string, ownerUser string, ownerGroup string, autoArchive bool, contactEmail string) (uuid.UUID, error) {
	taskId := uuid.New()
	transferObjects := map[string]interface{}{}

	client, err := globusauth.GetClientFromSession(ctx, i.globusAuthConf, i.sessionDuration)
	if err != nil {
		return uuid.UUID{}, err
	}

	// |-> globus dependencies
	// add transfer dependencies to the transferObjects map
	transferObjects["globus_client"] = client
	transferObjects["username"] = username

	err = i.taskQueue.AddTransferTask(datasetId, fileList, taskId, folderPath, ownerUser, ownerGroup, contactEmail, autoArchive, transferObjects)
	if err != nil {
		return uuid.UUID{}, err
	}
	return taskId, nil
}

func (i *IngestorWebServerImplemenation) addExtGlobusTransferTask(ctx context.Context, datasetId string, fileList []datasetIngestor.Datafile, autoArchive bool, scicatToken string) (string, error) {
	filesToTransfer := make([]extglobusservice.FileToTransfer, len(fileList))
	for i, file := range fileList {
		filesToTransfer[i].Path = file.Path
		filesToTransfer[i].IsSymlink = file.IsSymlink
	}
	return extglobusservice.RequestExternalTransferTask(
		ctx,
		i.taskQueue.Config.Transfer.ExtGlobus.TransferServiceUrl,
		scicatToken,
		i.taskQueue.Config.Transfer.ExtGlobus.SrcFacility,
		i.taskQueue.Config.Transfer.ExtGlobus.DstFacility,
		datasetId,
		&filesToTransfer,
	)
}

func (i *IngestorWebServerImplemenation) addS3TransferTask(ctx context.Context, datasetId string, fileList []datasetIngestor.Datafile, folderPath string, ownerUser string, ownerGroup string, autoArchive bool, contactEmail string, scicatToken string) (uuid.UUID, error) {
	taskId := uuid.New()
	transferObjects := map[string]interface{}{}

	// access and refresh token need be fetched at this point from the archiver backend since user token could expire
	accessToken, refreshToken, expires_in, err := s3upload.GetTokens(ctx, i.taskQueue.Config.Transfer.S3.Endpoint, scicatToken)
	if err != nil {
		return uuid.UUID{}, err
	}
	transferObjects["accessToken"] = accessToken
	transferObjects["refreshToken"] = refreshToken
	transferObjects["expires_in"] = expires_in

	err = i.taskQueue.AddTransferTask(datasetId, fileList, taskId, folderPath, ownerUser, ownerGroup, contactEmail, autoArchive, transferObjects)
	if err != nil {
		return uuid.UUID{}, err
	}
	return taskId, nil
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
