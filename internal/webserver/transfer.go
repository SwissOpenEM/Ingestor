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
	//deleteEntry := false
	//if request.Body.

	if i.taskQueue.GetTransferMethod() == transfertask.TransferExtGlobus {
		if request.Body.ScicatToken == nil {
			return TransferControllerDeleteTransfer400TextResponse("scicat token is required to process this request"), nil
		}
		delete := false
		if request.Body.DeleteTask != nil {
			delete = *request.Body.DeleteTask
		}
		err := extglobusservice.CancelTask(ctx, i.taskQueue.Config.Transfer.ExtGlobus.TransferServiceUrl, *request.Body.ScicatToken, request.Body.TransferId, delete)
		if err != nil {
			return TransferControllerDeleteTransfer400TextResponse(fmt.Sprintf("Couldn't cancel or delete task: %s", err.Error())), nil
		}
		status := "gone"
		return TransferControllerDeleteTransfer200JSONResponse{
			TransferId: request.Body.TransferId,
			Status:     &status,
		}, nil
	}

	uuid, err := uuid.Parse(request.Body.TransferId)
	if err != nil {
		return TransferControllerDeleteTransfer400TextResponse(fmt.Sprintf("Ingest ID '%s' could not be parsed as uuid: %s", request.Body.TransferId, err.Error())), nil
	}

	err = i.taskQueue.RemoveTask(uuid)
	if err != nil {
		return TransferControllerDeleteTransfer400TextResponse(err.Error()), nil
	}

	status := "gone"
	return TransferControllerDeleteTransfer200JSONResponse{
		TransferId: request.Body.TransferId,
		Status:     &status,
	}, nil
}

func (i *IngestorWebServerImplemenation) TransferControllerGetTransfer(ctx context.Context, request TransferControllerGetTransferRequestObject) (TransferControllerGetTransferResponseObject, error) {
	if request.Params.TransferId != nil {
		if i.taskQueue.GetTransferMethod() == transfertask.TransferExtGlobus {
			if request.Params.ScicatAPIToken == nil {
				return TransferControllerGetTransfer400TextResponse("no Scicat API token was provided"), nil
			}
			return GetTaskByJobIdFromScicat(i.taskQueue.Config.Scicat.Host, *request.Params.ScicatAPIToken, *request.Params.TransferId)
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
		if request.Params.ScicatAPIToken == nil {
			return TransferControllerGetTransfer400TextResponse("no Scicat API token was provided"), nil
		}
		return GetTasksFromScicat(i.taskQueue.Config.Scicat.Host, *request.Params.ScicatAPIToken, (page-1)*pageSize, pageSize)
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

func GetTasksFromScicat(scicatServer string, scicatToken string, skip uint, limit uint) (TransferControllerGetTransferResponseObject, error) {
	scicatUrl, _ := url.Parse(scicatServer)

	jobs, total, err := extglobusservice.GetGlobusTransferJobsFromScicat(fmt.Sprintf("%s://%s", scicatUrl.Scheme, scicatUrl.Host), scicatToken, skip, limit)
	if err != nil {
		return TransferControllerGetTransfer400TextResponse(err.Error()), nil
	}
	tasks := make([]TransferItem, len(jobs))
	for i, job := range jobs {
		tasks[i] = JobToTransferItem(job)
	}
	return TransferControllerGetTransfer200JSONResponse{
		Transfers: &tasks,
		Total:     getPointerOrNil(int(total)),
	}, nil
}

func JobToTransferItem(job jobs.ScicatJob) TransferItem {
	var status TransferItemStatus = InvalidStatus
	switch job.JobResultObject.Status {
	case jobs.Finished:
		status = Finished
	case jobs.Transferring:
		status = Transferring
	case jobs.Cancelled:
		status = Cancelled
	case jobs.Failed:
		status = Failed
	}
	return TransferItem{
		BytesTransferred: getPointerOrNil(int64(job.JobResultObject.BytesTransferred)),
		FilesTransferred: getPointerOrNil(int32(job.JobResultObject.FilesTransferred)),
		FilesTotal:       getPointerOrNil(int32(job.JobResultObject.FilesTotal)),
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
