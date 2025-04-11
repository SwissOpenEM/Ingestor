package webserver

import (
	"context"
	"fmt"
	"net/url"

	"github.com/SwissOpenEM/Ingestor/internal/extglobusservice"
	"github.com/SwissOpenEM/Ingestor/internal/transfertask"
	"github.com/SwissOpenEM/globus-transfer-service/jobs"
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
		if i.taskQueue.GetTransferMethod() == transfertask.TransferExtGlobus {
			return GetTaskByJobIdFromScicat(i.taskQueue.Config.Scicat.Host, "", *request.Params.TransferId)
		}
		id := *request.Params.TransferId
		uid, err := uuid.Parse(id)
		if err != nil {
			return TransferControllerGetTransfer400TextResponse(fmt.Sprintf("Can't parse UUID: %s", err.Error())), nil
		}

		status, err := i.taskQueue.GetTaskDetails(uid)
		if err != nil {
			return TransferControllerGetTransfer400TextResponse(fmt.Sprintf("No such task with id '%s'", uid.String())), nil
		}

		transferItems := []TransferItem{
			{
				TransferId:       id,
				Status:           statusToDto(status.Status),
				Message:          &status.Message,
				BytesTransferred: &status.BytesTransferred,
				BytesTotal:       &status.BytesTotal,
				FilesTransferred: &status.FilesTransferred,
				FilesTotal:       &status.FilesTotal,
			},
		}

		return TransferControllerGetTransfer200JSONResponse{
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

	if i.taskQueue.GetTransferMethod() == transfertask.TransferExtGlobus {
		return GetTasksFromScicat(i.taskQueue.Config.Scicat.Host, "", "", (page-1)*pageSize, page*pageSize)
	}

	resultNo := i.taskQueue.GetTaskCount()
	ids, statuses, err := i.taskQueue.GetTaskDetailsList((page-1)*pageSize, page*pageSize)
	if err != nil {
		return TransferControllerGetTransfer400TextResponse(err.Error()), nil
	}

	transferItems := []TransferItem{}
	for i, status := range statuses {
		idString := ids[i].String()
		transferItems = append(transferItems, TransferItem{
			TransferId: idString,
			Status:     statusToDto(status.Status),
			Message:    getPointerOrNil(status.Message),
		})
	}

	return TransferControllerGetTransfer200JSONResponse{
		Total:     &resultNo,
		Transfers: &transferItems,
	}, nil
}

func GetTaskByJobIdFromScicat(scicatServer string, scicatToken string, jobId string) (TransferControllerGetTransferResponseObject, error) {
	scicatUrl, _ := url.Parse(scicatServer)
	job, err := jobs.GetJobById(fmt.Sprintf("%s://%s", scicatUrl.Scheme, scicatUrl.Host), scicatToken, jobId)
	if err != nil {
		return TransferControllerGetTransfer400TextResponse(err.Error()), nil
	}
	return TransferControllerGetTransfer200JSONResponse{
		Transfers: &[]TransferItem{
			JobToTransferItem(job),
		},
	}, nil
}

func GetTasksFromScicat(scicatServer string, scicatToken string, ownerUser string, skip uint, limit uint) (TransferControllerGetTransferResponseObject, error) {
	scicatUrl, _ := url.Parse(scicatServer)
	jobs, err := extglobusservice.GetGlobusTransferJobsFromScicat(fmt.Sprintf("%s://%s", scicatUrl.Scheme, scicatUrl.Host), scicatToken, ownerUser, skip, limit)
	if err != nil {
		return TransferControllerGetTransfer400TextResponse(err.Error()), nil
	}
	tasks := make([]TransferItem, len(jobs))
	for i, job := range jobs {
		tasks[i] = JobToTransferItem(job)
	}
	return TransferControllerGetTransfer200JSONResponse{
		Transfers: &tasks,
		Total:     getPointerOrNil(len(tasks)),
	}, nil
}

func JobToTransferItem(job jobs.ScicatJob) TransferItem {
	var status TransferItemStatus
	if job.JobResultObject.Completed {
		status = Finished
	} else if job.JobResultObject.Error != "" {
		status = Failed
	} else {
		status = Transferring
	}
	return TransferItem{
		BytesTransferred: getPointerOrNil(int(job.JobResultObject.BytesTransferred)),
		FilesTransferred: getPointerOrNil(int(job.JobResultObject.FilesTransferred)),
		FilesTotal:       getPointerOrNil(int(job.JobResultObject.FilesTotal)),
		Status:           status,
		Message:          &job.StatusMessage,
		TransferId:       job.ID,
	}
}

func getPointerOrNil[T comparable](v T) *T {
	var a T
	if a == v {
		return nil
	} else {
		return &v
	}
}

func statusToDto(s transfertask.Status) TransferItemStatus {
	switch s {
	case transfertask.Waiting:
		return Waiting
	case transfertask.Transferring:
		return Transferring
	case transfertask.Finished:
		return Finished
	case transfertask.Failed:
		return Failed
	case transfertask.Cancelled:
		return Cancelled
	default:
		return InvalidStatus
	}
}
