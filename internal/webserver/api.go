//go:build go1.22

//go:generate go run github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen --config=cfg.yaml ../../api/openapi.yaml
//go:generate go run github.com/swaggo/swag/cmd/swag init -g api.go -o ../../docs
package webserver

import (
	"encoding/json"
	"io"
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
		c.JSON(http.StatusBadRequest, gin.H{"error": "Failed to read request body"})
		return
	}
	err = json.Unmarshal(reqBody, &request)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid JSON data"})
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
		c.JSON(http.StatusBadRequest, gin.H{"error": "Metadata is not a valid JSON document."})
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
	var result IngestorUiDeleteTransferResponse

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

	c.JSON(http.StatusOK, result)
}
