package webserver

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"slices"

	"github.com/SwissOpenEM/Ingestor/internal/core"
	"github.com/SwissOpenEM/Ingestor/internal/s3upload"
	"github.com/SwissOpenEM/Ingestor/internal/task"
	"github.com/SwissOpenEM/Ingestor/internal/webserver/globusauth"
	"github.com/google/uuid"
	"github.com/paulscherrerinstitute/scicat-cli/v3/datasetIngestor"
)

func (i *IngestorWebServerImplemenation) DatasetControllerBrowseFilesystem(ctx context.Context, request DatasetControllerBrowseFilesystemRequestObject) (DatasetControllerBrowseFilesystemResponseObject, error) {
	// an internal function used to determine if a folder has subfolders
	folderHasFilesOrSubFolders := func(path string) (bool, bool) {
		folder, err := os.Open(path)
		if err != nil {
			return false, false
		}
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

	// get absolute path in os format
	// Clean = remove relative paths and convert / to \ under Windows (consistent pathstrings across OS's)
	absPath := filepath.Join(i.pathConfig.CollectionLocation, filepath.Clean(request.Params.Path))

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
	err = filepath.WalkDir(absPath, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil // skip errored elements
		}
		if d.IsDir() && path != absPath {
			if folderCounter >= start && folderCounter < end {
				hasFiles, hasChildren := folderHasFilesOrSubFolders(path)
				relativePath, _ := filepath.Rel(i.pathConfig.CollectionLocation, path)

				folders[folderCounter-start].Name = d.Name()
				folders[folderCounter-start].Path = "/" + relativePath
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

	fp, ok := metadata["sourceFolder"]
	if !ok {
		return DatasetControllerIngestDataset400TextResponse("sourceFolder is not present in the metadata"), nil
	}
	folderPath, ok := fp.(string)
	if !ok {
		return DatasetControllerIngestDataset400TextResponse("sourceFolder is not a string"), nil
	}
	folderPath = filepath.Join(i.pathConfig.CollectionLocation, filepath.Clean(folderPath))

	// check if folder exists
	err = core.CheckIfFolderExists(folderPath)
	if err != nil {
		return DatasetControllerIngestDataset400TextResponse(fmt.Sprintf("dataset location lookup error: %s", err.Error())), nil
	}

	// do catalogue insertion
	datasetId, totalSize, fileList, err := core.AddDatasetToScicat(metadata, folderPath, request.Body.UserToken, i.taskQueue.Config.Scicat.Host)
	if err != nil {
		return DatasetControllerIngestDataset400TextResponse(err.Error()), nil
	}

	taskId, err := i.addTransferTask(ctx, datasetId, fileList, totalSize, metadata, request)

	if err != nil {
		if _, ok := err.(*os.PathError); ok {
			return nil, fmt.Errorf("could not create the task due to a path error: %s", err.Error())
		} else {
			return DatasetControllerIngestDataset400TextResponse(fmt.Sprintf("You don't have permissions to access the dataset folder or it doesn't exist: %s", err.Error())), nil
		}
	}
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

func (i *IngestorWebServerImplemenation) addTransferTask(ctx context.Context, datasetId string, fileList []datasetIngestor.Datafile, totalSize int64, metadata map[string]interface{}, request DatasetControllerIngestDatasetRequestObject) (uuid.UUID, error) {
	taskId := uuid.New()
	transferObjects := map[string]interface{}{}
	if i.taskQueue.GetTransferMethod() == task.TransferGlobus {
		client, err := globusauth.GetClientFromSession(ctx, i.globusAuthConf, i.sessionDuration)
		if err != nil {
			return uuid.UUID{}, err
		}
		// |-> globus dependencies
		// add transfer dependencies to the transferObjects map
		transferObjects["globus_client"] = client

	} else if i.taskQueue.GetTransferMethod() == task.TransferS3 {

		// access and refresh token need be fetched at this point from the archiver backend since user token could expire
		accessToken, refreshToken, err := s3upload.GetTokens(ctx, i.taskQueue.Config.Transfer.S3.Endpoint, request.Body.UserToken)
		if err != nil {
			return uuid.UUID{}, err
		}
		transferObjects["accessToken"] = accessToken
		transferObjects["refreshToken"] = refreshToken
	}
	err := i.taskQueue.AddTransferTask(transferObjects, datasetId, fileList, totalSize, metadata, taskId)
	if err != nil {
		return uuid.UUID{}, err
	}
	return taskId, nil
}

func (i *IngestorWebServerImplemenation) DatasetControllerGetDataset(ctx context.Context, request DatasetControllerGetDatasetRequestObject) (DatasetControllerGetDatasetResponseObject, error) {
	files, err := os.ReadDir(i.pathConfig.CollectionLocation)
	if err != nil {
		return nil, err
	}

	var datasets []string
	for _, file := range files {
		if file.IsDir() {
			datasets = append(datasets, file.Name())
		}
	}
	slices.Sort(datasets)

	page := uint(1)
	pageSize := uint(10)
	if request.Params.Page != nil {
		page = max(*request.Params.Page, 1)
	}
	if request.Params.PageSize != nil {
		pageSize = min(*request.Params.PageSize, 100)
	}

	return DatasetControllerGetDataset200JSONResponse{
		Datasets: safeSubslice(datasets, (page-1)*pageSize, page*pageSize),
		Total:    len(datasets),
	}, nil
}

//func ptr[T any](v T) *T {
//	var temp T = v
//	return &temp
//}

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
