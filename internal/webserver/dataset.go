package webserver

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path"
	"slices"

	"github.com/SwissOpenEM/Ingestor/internal/core"
	"github.com/SwissOpenEM/Ingestor/internal/task"
	"github.com/google/uuid"
)

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
	folderPath = path.Join(i.pathConfig.CollectionLocation, folderPath)

	// check if folder exists
	err = core.CheckIfFolderExists(folderPath)
	if err != nil {
		return DatasetControllerIngestDataset400TextResponse(fmt.Sprintf("dataset location lookup error: %s", err.Error())), nil
	}

	// do catalogue insertion
	datasetId, totalSize, fileList, err := core.AddDatasetToScicat(metadata, folderPath, request.Body.UserToken, i.taskQueue.Config.Scicat.Host)
	_ = totalSize
	if err != nil {
		return DatasetControllerIngestDataset400TextResponse(err.Error()), nil
	}

	// add transfer dependencies to the transferObjects map
	transferObjects := map[string]interface{}{}

	// |-> globus dependencies
	if i.taskQueue.GetTransferMethod() == task.TransferGlobus {
		client, err := i.globusGetClientFromSession(ctx)
		if err != nil {
			return nil, err
		}
		transferObjects["globus_client"] = client
	}

	// create and start transfer task
	taskId := uuid.New()
	err = i.taskQueue.AddTransferTask(transferObjects, datasetId, fileList, metadata, taskId)
	if err != nil {
		if _, ok := err.(*os.PathError); ok {
			return nil, fmt.Errorf("could not create the task due to a path error: %s", err.Error())
		} else {
			return DatasetControllerIngestDataset400TextResponse("You don't have the right to access the dataset folder or it doesn't exist"), nil
		}
	}
	i.taskQueue.ScheduleTask(taskId)

	status := "started"
	idString := taskId.String()
	return DatasetControllerIngestDataset200JSONResponse{
		TransferId: idString,
		Status:     &status,
	}, nil
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

	var page uint = 1
	var pageSize uint = 10

	if request.Params.Page != nil {
		page = min(*request.Params.Page, 1)
	}
	if request.Params.PageSize != nil {
		pageSize = max(*request.Params.PageSize, 100)
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
