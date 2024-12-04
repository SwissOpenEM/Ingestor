package webserver

import (
	"context"
	"fmt"
	"math"

	"github.com/google/uuid"
)

// TransferControllerDeleteTransfer implements ServerInterface.
//
// @Description	Cancel a data transfer
// @Tags            transfer
// @Accept          json
// @Produce		    json
// @Param           request   body     webserver.DeleteTransferRequest true "it contains the id to cancel"
// @Success         200       {object} webserver.TransferControllerDeleteTransfer200JSONResponse "returns the status and id of the affected task"
// @Failure         400       {string} string                                                    "invalid request"
// @Router          /transfer [delete]
func (i *IngestorWebServerImplemenation) TransferControllerDeleteTransfer(ctx context.Context, request TransferControllerDeleteTransferRequestObject) (TransferControllerDeleteTransferResponseObject, error) {
	if request.Body.IngestId == nil {
		return TransferControllerDeleteTransfer400TextResponse("Ingest ID was not specified in the request"), nil
	}

	id := *request.Body.IngestId
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
		IngestId: &id,
		Status:   &status,
	}, nil
}

// TransferControllerGetTransfer implements ServerInterface.
//
// @Description	"Get list of transfers. Optional use the transferId parameter to only get one item."
// @Tags	        transfer
// @Produce         json
// @Param           page       query    int                                            false                           "page of transfers"
// @Param           pageSize   query    int                                            false                           "number of elements per page"
// @Param           transferId query    int                                            false                           "get specific transfer by id"
// @Success         200        {object} webserver.TransferControllerGetTransfer200JSONResponse   "returns the list of transfers"
// @Failure         400        {string} string                                                   "the request is invalid"
// @Router          /transfer  [get]
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
				Status:     &status.StatusMessage,
				TransferId: &id,
			},
		}

		totalItems := len(transferItems)
		return TransferControllerGetTransfer200JSONResponse{
			Total:     &totalItems,
			Transfers: &transferItems,
		}, nil
	}

	if request.Params.Page != nil {
		var start, end, pageIndex, pageSize uint

		pageSize = 50
		if request.Params.PageSize != nil {
			pageSize = uint(*request.Params.PageSize)
		}

		if *request.Params.Page <= 0 {
			pageIndex = 1
		} else {
			pageIndex = uint(*request.Params.Page)
		}

		start = (pageIndex - 1) * pageSize
		end = pageIndex * pageSize

		resultNo := i.taskQueue.GetTaskCount()
		ids, statuses, err := i.taskQueue.GetTaskStatusList(start, end)
		if err != nil {
			return TransferControllerGetTransfer400TextResponse(err.Error()), nil
		}

		transferItems := []TransferItem{}
		for i, status := range statuses {
			idString := ids[i].String()
			s := status.StatusMessage
			if !status.Failed {
				if status.Finished {
					s = "finished"
				} else if status.Started {
					s = fmt.Sprintf(
						"progress: %d%%",
						int(math.Round(float64(status.BytesTransferred)/float64(status.BytesTotal))),
					)
				} else {
					s = "queued"
				}
			} else if status.StatusMessage == "" {
				s = "failed - unknown error"
			}
			transferItems = append(transferItems, TransferItem{
				Status:     &s,
				TransferId: &idString,
			})
		}

		return TransferControllerGetTransfer200JSONResponse{
			Total:     &resultNo,
			Transfers: &transferItems,
		}, nil
	}

	return TransferControllerGetTransfer400TextResponse("Not enough parameters"), nil
}
