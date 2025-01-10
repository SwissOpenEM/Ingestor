package webserver

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path"
	"slices"

	"github.com/google/uuid"
)

func (i *IngestorWebServerImplemenation) DatasetControllerIngestDataset(ctx context.Context, request DatasetControllerIngestDatasetRequestObject) (DatasetControllerIngestDatasetResponseObject, error) {
	// get sourcefolder from metadata
	var metadata map[string]interface{}
	err := json.Unmarshal([]byte(request.Body.MetaData), &metadata)
	if err != nil {
		return DatasetControllerIngestDataset400TextResponse(err.Error()), nil
	}

	// add collection location
	dsPath, ok := metadata["sourceFolder"].(string)
	if !ok {
		return DatasetControllerIngestDataset400TextResponse("datasetFolder is not a string"), nil
	}
	metadata["sourceFolder"] = path.Join(i.pathConfig.CollectionLocation, dsPath)

	// create and start task
	id := uuid.New()
	err = i.taskQueue.CreateTaskFromMetadata(id, metadata)
	if err != nil {
		if _, ok := err.(*os.PathError); ok {
			return nil, fmt.Errorf("could not create the task due to a path error: %s", err.Error())
		} else {
			return DatasetControllerIngestDataset400TextResponse("You don't have the right to access the dataset folder or it doesn't exist"), nil
		}
	}
	i.taskQueue.ScheduleTask(id)

	// NOTE: because of the way the tasks are created, right now it'll search for a metadata.json
	//   in the dataset folder to get the metadata, we can't pass on the one we got through this
	//   request
	// TODO: change this so that a task will accept a struct containing the dataset
	status := "started"
	idString := id.String()
	return DatasetControllerIngestDataset200JSONResponse{
		IngestId: &idString,
		Status:   &status,
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
