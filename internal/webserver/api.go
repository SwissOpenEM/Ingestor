//go:build go1.22

//go:generate go run github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen --config=cfg.yaml ../../api/openapi.yaml
//go:generate go run github.com/swaggo/swag/cmd/swag init -g api.go -o ../../docs
package webserver

import (
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"

	"github.com/SwissOpenEM/Ingestor/internal/core"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

var _ ServerInterface = (*IngestorWebServerImplemenation)(nil)

type IngestorWebServerImplemenation struct {
	version   string
	taskQueue *core.TaskQueue
}

//	@contact.name	SwissOpenEM
//	@contact.url	https://swissopenem.github.io
//	@contact.email	spencer.bliven@psi.ch

// @license.name	Apache 2.0
// @license.url	http://www.apache.org/licenses/LICENSE-2.0.html

func NewIngestorWebServer(version string, taskQueue *core.TaskQueue) *IngestorWebServerImplemenation {
	return &IngestorWebServerImplemenation{version: version, taskQueue: taskQueue}
}

// DatasetControllerIngestDataset implements ServerInterface.
//
//	@Description	Ingest a new dataset
//	@Tags			datasets
//	@Accept			json
//	@Produce		json
//
//	@Router			/datasets [post]
func (i *IngestorWebServerImplemenation) DatasetControllerIngestDataset(c *gin.Context) {
	var request IngestorUiPostDatasetRequest
	var result IngestorUiPostDatasetResponse

	// convert body to struct
	reqBody, err := io.ReadAll(c.Request.Body)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Failed to read request body: %v", err)})
		return
	}
	err = json.Unmarshal(reqBody, &request)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Invalid JSON data: %v", err)})
		return
	}
	if request.MetaData == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Metadata is empty"})
		return
	}

	// get sourcefolder from metadata
	metadataString := *request.MetaData
	var metadata map[string]interface{}
	err = json.Unmarshal([]byte(metadataString), &metadata)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Metadata is not a valid JSON document: %v", err)})
		return
	}

	// create and start task
	id := uuid.New()
	i.taskQueue.CreateTaskFromMetadata(id, metadata)
	i.taskQueue.ScheduleTask(id)

	// NOTE: because of the way the tasks are created, right now it'll search for a metadata.json
	//   in the dataset folder to get the metadata, we can't pass on the one we got through this
	//   request
	// TODO: change this so that a task will accept a struct containing the dataset
	status := "started"
	idString := id.String()
	result.IngestId = &idString
	result.Status = &status
	c.JSON(http.StatusOK, result)
}

// OtherControllerGetVersion implements ServerInterface.
//
//	@Description	Get the used ingestor version
//	@Tags			other
//	@Accept			json
//	@Produce		json
//
//	@Router			/version [get]
func (i *IngestorWebServerImplemenation) OtherControllerGetVersion(c *gin.Context) {
	var result IngestorUiOtherVersionResponse
	result.Version = &i.version

	c.JSON(http.StatusOK, result)
}

// TransferControllerDeleteTransfer implements ServerInterface.
//
//	@Description	Cancel a data transfer
//	@Tags			transfer
//	@Accept			json
//	@Produce		json
//
//	@Router			/transfer [delete]
func (i *IngestorWebServerImplemenation) TransferControllerDeleteTransfer(c *gin.Context) {
	var request IngestorUiDeleteTransferRequest
	var result IngestorUiDeleteTransferResponse

	reqBody, err := io.ReadAll(c.Request.Body)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Failed to read request body: %v", err)})
		return
	}
	err = json.Unmarshal(reqBody, &request)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Invalid JSON data: %v", err)})
		return
	}
	if request.IngestId == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Ingest ID was not specified in the request"})
		return
	}

	id := *request.IngestId
	uuid, err := uuid.Parse(id)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Ingest ID is not a valid uuid: %v", err)})
		return
	}

	err = i.taskQueue.RemoveTask(uuid)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Task not found"})
		return
	}

	c.JSON(http.StatusOK, result)
}

// TransferControllerGetTransfer implements ServerInterface.
//
//	@Description	Get list of transfers. Optional use the transferId parameter to only get one item.
//	@Tags			transfer
//	@Accept			json
//	@Produce		json
//	@param			params	path	TransferControllerGetTransferParams	true	"params"
//
//	@Router			/transfer [get]
func (i *IngestorWebServerImplemenation) TransferControllerGetTransfer(c *gin.Context, params TransferControllerGetTransferParams) {
	var result IngestorUiGetTransferResponse

	if params.TransferId != nil {
		id := *params.TransferId
		uid, err := uuid.Parse(id)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Can't parse UUID: %v", err)})
			return
		}

		status, err := i.taskQueue.GetTaskStatus(uid)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("No such task with id '%s'", uid.String())})
			return
		}
		transferItems := []IngestorUiGetTransferItem{
			{
				Status:     &status.StatusMessage,
				TransferId: &id,
			},
		}

		result.Transfers = &transferItems

		c.JSON(http.StatusOK, result)
		return
	}

	if params.Page != nil {
		var start, end, pageIndex, pageSize uint

		pageSize = 50
		if params.PageSize != nil {
			pageSize = uint(*params.PageSize)
		}

		if *params.Page <= 0 {
			pageIndex = 1
		} else {
			pageIndex = uint(*params.Page)
		}

		start = (pageIndex - 1) * pageSize
		end = pageIndex * pageSize

		resultNo := i.taskQueue.GetTaskCount()
		ids, statuses, err := i.taskQueue.GetTaskStatusList(start, end)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		transferItems := []IngestorUiGetTransferItem{}
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
			transferItems = append(transferItems, IngestorUiGetTransferItem{
				Status:     &s,
				TransferId: &idString,
			})
		}

		result.Total = &resultNo
		result.Transfers = &transferItems
		c.JSON(http.StatusOK, result)
		return
	}

	c.JSON(http.StatusBadRequest, gin.H{"error": "Not enough parameters"})
}
