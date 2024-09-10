//go:build go1.22

//go:generate go run github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen --config=cfg.yaml ../../api/openapi.yaml
package webserver

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

var _ ServerInterface = (*IngestorWebServerImplemenation)(nil)

type IngestorWebServerImplemenation struct {
	version string
}

func NewIngestorWebServer(version string) *IngestorWebServerImplemenation {
	return &IngestorWebServerImplemenation{version: version}
}

// DatasetControllerIngestDataset implements ServerInterface.
func (i *IngestorWebServerImplemenation) DatasetControllerIngestDataset(c *gin.Context) {
	var result IngestorUiPostDatasetResponse

	c.JSON(http.StatusOK, result)
}

// OtherControllerGetVersion implements ServerInterface.
func (i *IngestorWebServerImplemenation) OtherControllerGetVersion(c *gin.Context) {
	var result IngestorUiOtherVersionResponse
	result.Version = &i.version

	c.JSON(http.StatusOK, result)
}

// TransferControllerDeleteTransfer implements ServerInterface.
func (i *IngestorWebServerImplemenation) TransferControllerDeleteTransfer(c *gin.Context) {
	var result IngestorUiDeleteTransferResponse

	c.JSON(http.StatusOK, result)
}

// TransferControllerGetTransfer implements ServerInterface.
func (i *IngestorWebServerImplemenation) TransferControllerGetTransfer(c *gin.Context, params TransferControllerGetTransferParams) {
	var result IngestorUiGetTransferResponse

	c.JSON(http.StatusOK, result)
}
