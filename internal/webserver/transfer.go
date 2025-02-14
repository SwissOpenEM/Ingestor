package webserver

import (
	"context"
	"fmt"

	"github.com/google/uuid"
)

func (i *IngestorWebServerImplemenation) TransferControllerDeleteTransfer(ctx context.Context, request TransferControllerDeleteTransferRequestObject) (TransferControllerDeleteTransferResponseObject, error) {
	id := request.Body.TransferId
	uuid, err := uuid.Parse(id)
	if err != nil {
		return TransferControllerDeleteTransfer400TextResponse(fmt.Sprintf("Ingest ID '%s' could not be parsed as uuid: %s", id, err.Error())), nil
	}

	err = i.taskQueue.RemoveTask(uuid)
	if err != nil {
		return TransferControllerDeleteTransfer400TextResponse(err.Error()), nil
	}

	status := "gone"
	return TransferControllerDeleteTransfer200JSONResponse{
		TransferId: id,
		Status:     &status,
	}, nil
}

func (i *IngestorWebServerImplemenation) TransferControllerGetTransfer(ctx context.Context, request TransferControllerGetTransferRequestObject) (TransferControllerGetTransferResponseObject, error) {
	if request.Params.TransferId != nil {
		id := *request.Params.TransferId
		uid, err := uuid.Parse(id)
		if err != nil {
			return TransferControllerGetTransfer400TextResponse(fmt.Sprintf("Can't parse UUID: %s", err.Error())), nil
		}

		status, err := i.taskQueue.GetTaskStatus(uid)
		if err != nil {
			return TransferControllerGetTransfer400TextResponse(fmt.Sprintf("No such task with id '%s'", uid.String())), nil
		}
		transferItems := []TransferItem{
			{
				TransferId:       id,
				Status:           &status.StatusMessage,
				Started:          &status.Started,
				Finished:         &status.Finished,
				Failed:           &status.Failed,
				BytesTransferred: &status.BytesTransferred,
				BytesTotal:       &status.BytesTotal,
				FilesTransferred: &status.FilesTransferred,
				FilesTotal:       &status.FilesTotal,
			},
		}

		totalItems := len(transferItems)
		return TransferControllerGetTransfer200JSONResponse{
			Total:     &totalItems,
			Transfers: &transferItems,
		}, nil
	}

	page := uint(1)
	pageSize := uint(10)
	if request.Params.Page != nil {
		page = max(*request.Params.Page, 1)
	}
	if request.Params.PageSize != nil {
		pageSize = min(*request.Params.PageSize, 100)
	}

	resultNo := i.taskQueue.GetTaskCount()
	ids, statuses, err := i.taskQueue.GetTaskStatusList((page-1)*pageSize, page*pageSize)
	if err != nil {
		return TransferControllerGetTransfer400TextResponse(err.Error()), nil
	}

	transferItems := []TransferItem{}
	for i, status := range statuses {
		idString := ids[i].String()
		transferItems = append(transferItems, TransferItem{
			TransferId: idString,
			Status:     getPointerOrNil(status.StatusMessage),
			Started:    getPointerOrNil(status.Started),
			Finished:   getPointerOrNil(status.Finished),
			Failed:     getPointerOrNil(status.Failed),
		})
	}

	return TransferControllerGetTransfer200JSONResponse{
		Total:     &resultNo,
		Transfers: &transferItems,
	}, nil
}

func getPointerOrNil[T comparable](v T) *T {
	var a T
	if a == v {
		return nil
	} else {
		return &v
	}
}
