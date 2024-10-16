// Package webserver provides primitives to interact with the openapi HTTP API.
//
// Code generated by github.com/oapi-codegen/oapi-codegen/v2 version v2.3.0 DO NOT EDIT.
package webserver

import (
	"bytes"
	"compress/gzip"
	"encoding/base64"
	"fmt"
	"net/http"
	"net/url"
	"path"
	"strings"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/gin-gonic/gin"
	"github.com/oapi-codegen/runtime"
)

// IngestorUiDeleteTransferRequest defines model for IngestorUiDeleteTransferRequest.
type IngestorUiDeleteTransferRequest struct {
	// IngestId Ingestion id to abort the ingestion
	IngestId *string `json:"ingestId,omitempty"`
}

// IngestorUiDeleteTransferResponse defines model for IngestorUiDeleteTransferResponse.
type IngestorUiDeleteTransferResponse struct {
	// IngestId Ingestion id to abort the ingestion
	IngestId *string `json:"ingestId,omitempty"`

	// Status New status of the ingestion.
	Status *string `json:"status,omitempty"`
}

// IngestorUiGetTransferItem defines model for IngestorUiGetTransferItem.
type IngestorUiGetTransferItem struct {
	Status     *string `json:"status,omitempty"`
	TransferId *string `json:"transferId,omitempty"`
}

// IngestorUiGetTransferResponse defines model for IngestorUiGetTransferResponse.
type IngestorUiGetTransferResponse struct {
	// Total Total number of transfers.
	Total     *int                         `json:"total,omitempty"`
	Transfers *[]IngestorUiGetTransferItem `json:"transfers,omitempty"`
}

// IngestorUiInvalidRequestResponse defines model for IngestorUiInvalidRequestResponse.
type IngestorUiInvalidRequestResponse struct {
	// Message Error message describing the invalid request.
	Message *string `json:"message,omitempty"`
}

// IngestorUiOtherVersionResponse defines model for IngestorUiOtherVersionResponse.
type IngestorUiOtherVersionResponse struct {
	// Version Version of the ingestor.
	Version *string `json:"version,omitempty"`
}

// IngestorUiPostDatasetRequest defines model for IngestorUiPostDatasetRequest.
type IngestorUiPostDatasetRequest struct {
	// MetaData The metadata of the dataset.
	MetaData *string `json:"metaData,omitempty"`
}

// IngestorUiPostDatasetResponse defines model for IngestorUiPostDatasetResponse.
type IngestorUiPostDatasetResponse struct {
	// IngestId The unique ingestion id of the dataset.
	IngestId *string `json:"ingestId,omitempty"`

	// Status The status of the ingestion. Can be used to send a message back to the ui.
	Status *string `json:"status,omitempty"`
}

// TransferControllerGetTransferParams defines parameters for TransferControllerGetTransfer.
type TransferControllerGetTransferParams struct {
	TransferId *string `form:"transferId,omitempty" json:"transferId,omitempty"`
	Page       *int    `form:"page,omitempty" json:"page,omitempty"`
	PageSize   *int    `form:"pageSize,omitempty" json:"pageSize,omitempty"`
}

// DatasetControllerIngestDatasetJSONRequestBody defines body for DatasetControllerIngestDataset for application/json ContentType.
type DatasetControllerIngestDatasetJSONRequestBody = IngestorUiPostDatasetRequest

// TransferControllerDeleteTransferJSONRequestBody defines body for TransferControllerDeleteTransfer for application/json ContentType.
type TransferControllerDeleteTransferJSONRequestBody = IngestorUiDeleteTransferRequest

// ServerInterface represents all server handlers.
type ServerInterface interface {
	// Ingest a new dataset
	// (POST /dataset)
	DatasetControllerIngestDataset(c *gin.Context)
	// Cancel a data transfer
	// (DELETE /transfer)
	TransferControllerDeleteTransfer(c *gin.Context)
	// Get list of transfers. Optional use the transferId parameter to only get one item.
	// (GET /transfer)
	TransferControllerGetTransfer(c *gin.Context, params TransferControllerGetTransferParams)
	// Get the used ingestor version
	// (GET /version)
	OtherControllerGetVersion(c *gin.Context)
}

// ServerInterfaceWrapper converts contexts to parameters.
type ServerInterfaceWrapper struct {
	Handler            ServerInterface
	HandlerMiddlewares []MiddlewareFunc
	ErrorHandler       func(*gin.Context, error, int)
}

type MiddlewareFunc func(c *gin.Context)

// DatasetControllerIngestDataset operation middleware
func (siw *ServerInterfaceWrapper) DatasetControllerIngestDataset(c *gin.Context) {

	for _, middleware := range siw.HandlerMiddlewares {
		middleware(c)
		if c.IsAborted() {
			return
		}
	}

	siw.Handler.DatasetControllerIngestDataset(c)
}

// TransferControllerDeleteTransfer operation middleware
func (siw *ServerInterfaceWrapper) TransferControllerDeleteTransfer(c *gin.Context) {

	for _, middleware := range siw.HandlerMiddlewares {
		middleware(c)
		if c.IsAborted() {
			return
		}
	}

	siw.Handler.TransferControllerDeleteTransfer(c)
}

// TransferControllerGetTransfer operation middleware
func (siw *ServerInterfaceWrapper) TransferControllerGetTransfer(c *gin.Context) {

	var err error

	// Parameter object where we will unmarshal all parameters from the context
	var params TransferControllerGetTransferParams

	// ------------- Optional query parameter "transferId" -------------

	err = runtime.BindQueryParameter("form", true, false, "transferId", c.Request.URL.Query(), &params.TransferId)
	if err != nil {
		siw.ErrorHandler(c, fmt.Errorf("Invalid format for parameter transferId: %w", err), http.StatusBadRequest)
		return
	}

	// ------------- Optional query parameter "page" -------------

	err = runtime.BindQueryParameter("form", true, false, "page", c.Request.URL.Query(), &params.Page)
	if err != nil {
		siw.ErrorHandler(c, fmt.Errorf("Invalid format for parameter page: %w", err), http.StatusBadRequest)
		return
	}

	// ------------- Optional query parameter "pageSize" -------------

	err = runtime.BindQueryParameter("form", true, false, "pageSize", c.Request.URL.Query(), &params.PageSize)
	if err != nil {
		siw.ErrorHandler(c, fmt.Errorf("Invalid format for parameter pageSize: %w", err), http.StatusBadRequest)
		return
	}

	for _, middleware := range siw.HandlerMiddlewares {
		middleware(c)
		if c.IsAborted() {
			return
		}
	}

	siw.Handler.TransferControllerGetTransfer(c, params)
}

// OtherControllerGetVersion operation middleware
func (siw *ServerInterfaceWrapper) OtherControllerGetVersion(c *gin.Context) {

	for _, middleware := range siw.HandlerMiddlewares {
		middleware(c)
		if c.IsAborted() {
			return
		}
	}

	siw.Handler.OtherControllerGetVersion(c)
}

// GinServerOptions provides options for the Gin server.
type GinServerOptions struct {
	BaseURL      string
	Middlewares  []MiddlewareFunc
	ErrorHandler func(*gin.Context, error, int)
}

// RegisterHandlers creates http.Handler with routing matching OpenAPI spec.
func RegisterHandlers(router gin.IRouter, si ServerInterface) {
	RegisterHandlersWithOptions(router, si, GinServerOptions{})
}

// RegisterHandlersWithOptions creates http.Handler with additional options
func RegisterHandlersWithOptions(router gin.IRouter, si ServerInterface, options GinServerOptions) {
	errorHandler := options.ErrorHandler
	if errorHandler == nil {
		errorHandler = func(c *gin.Context, err error, statusCode int) {
			c.JSON(statusCode, gin.H{"msg": err.Error()})
		}
	}

	wrapper := ServerInterfaceWrapper{
		Handler:            si,
		HandlerMiddlewares: options.Middlewares,
		ErrorHandler:       errorHandler,
	}

	router.POST(options.BaseURL+"/dataset", wrapper.DatasetControllerIngestDataset)
	router.DELETE(options.BaseURL+"/transfer", wrapper.TransferControllerDeleteTransfer)
	router.GET(options.BaseURL+"/transfer", wrapper.TransferControllerGetTransfer)
	router.GET(options.BaseURL+"/version", wrapper.OtherControllerGetVersion)
}

// Base64 encoded, gzipped, json marshaled Swagger object
var swaggerSpec = []string{

	"H4sIAAAAAAAC/9RYTXPbNhD9Kztoj6zkND3pljppRofGnnz0kvqwIlcSUhBggKVUNaP/3lkQFEmTcux6",
	"3DY3GSQWb3ffWzz6i8pdWTlLloNafFEh31KJ8efSbiiw8x/0SzLE9N6jDWvyb+lzTYHllcq7ijxriht0",
	"3LAs5HdBIfe6Yu2sWqRQ2lnQBbADXDnPwFsC3T5RmeJDRWqhAnttN+p4PK241SfKWR2zOzCFytlATw4q",
	"U4GR6zAO94b20DwDtx6GmT00udfEbWZLpnKcVQdihI/bjcXE43sfe76g7BjNOP33sgy2LlfkYwFSoNBL",
	"XlumDfk+yqZHTGX88b2ntVqo7+YdKeeJkfPz1emSQu/x8LUsl3aHRheJxucTLSkE3NA41VfeOw/pMTTP",
	"VtpuUtNjdPBN+Ae3/oq35H8jH7Sz57HtmhfG2NLOIQWdfzCMaxf4JTIG4rN6L4lR3pkgw5ZAnhbI2EIp",
	"mmiPQ/JwlQuU2urPdU+PIvivojqvdAl5TulwiRZWBHWgOFQC2QLwRJYV5n/Ismyq9b1qIUvart0Yx1sK",
	"DC+ul7B2HnJXlrXVOcYEV8R7IhvPaUsJH5aAtuj+FjACL5Df6ZxmIIndWuzFpQB7zdsY89WvP7zL9SVy",
	"t/l3K9sFDhrj9iG+18pcCiWlzgBrdiWyzjuC0J/sMY+4BV8qbVuyzzV5Tc0Y0WykNOnoUyIvrpcq6zSh",
	"ns0uZhfSQVeRxUqrhXo+u5g9V5mqkLexpfPU+cgl19B7WN5LT8gE2HIEyFBJlkFbSAgiXKlWSyXpbfNM",
	"8ApDYz+EmCqR+NJZ9s4Y8g3+tKwylUbGz644CJrcWSYbgWFVmdTa+afQ6L6Zi/efmhOKjtySU7WnQi3Y",
	"1xQXGpXFMv14cfHUWJKmI5hhB9IrPeGGOs8phHVtpL0/PQm4M/fDBL7lcNZH9Ya6LNEfTv4CECztWw4J",
	"iXET1OKjalduZNe8FUpDQ7E28mtIoPbS6xg0NEFPTqFpH/ifseiMBZxoVPsO5GhzMlT0iGQO/38qXUbY",
	"aRSdhmqPTKelm2OmNsRTlwV7TTuZZxVutEWmAowOPDBrzYR3cRMaWGvDJBfTeJqNydizZXHQeiyJo8P7",
	"KBe0WigZ5QeVKYuljPGeU816pR3diNO7K/Fm/X3DfK/l7kh2VO7HlPTQjZ8M6V2HvNN/3XXQm7HlhYri",
	"gTR51M0jxeEsXa2bov5j39xR8bZ3zh4V7WaC2q+J4STAdLn7xMZvTYiSy0g0M7hqBVMHGvieZQEnIYg3",
	"cNYcYEMMzhJI+2bTIpYroefx71a0HJjX3os32Z33/0P9xu+MgXjTp4P6V2b35GfOZDuSx2sT+5aZE12/",
	"fBjoW1n1SOCkMKrRkXjrdoDW3qiF2jJXYTGfY6VnxuVoti7wfPdMyVRJIW7z5Krtu8jOxLnPrjUkoZt1",
	"rSGREXDfCJ0IxmN9ItAvtZf0wHUBZTZvyJJHA/KZ40tM/3lJ4ZqCHG+OfwcAAP//2ANJ1qsSAAA=",
}

// GetSwagger returns the content of the embedded swagger specification file
// or error if failed to decode
func decodeSpec() ([]byte, error) {
	zipped, err := base64.StdEncoding.DecodeString(strings.Join(swaggerSpec, ""))
	if err != nil {
		return nil, fmt.Errorf("error base64 decoding spec: %w", err)
	}
	zr, err := gzip.NewReader(bytes.NewReader(zipped))
	if err != nil {
		return nil, fmt.Errorf("error decompressing spec: %w", err)
	}
	var buf bytes.Buffer
	_, err = buf.ReadFrom(zr)
	if err != nil {
		return nil, fmt.Errorf("error decompressing spec: %w", err)
	}

	return buf.Bytes(), nil
}

var rawSpec = decodeSpecCached()

// a naive cached of a decoded swagger spec
func decodeSpecCached() func() ([]byte, error) {
	data, err := decodeSpec()
	return func() ([]byte, error) {
		return data, err
	}
}

// Constructs a synthetic filesystem for resolving external references when loading openapi specifications.
func PathToRawSpec(pathToFile string) map[string]func() ([]byte, error) {
	res := make(map[string]func() ([]byte, error))
	if len(pathToFile) > 0 {
		res[pathToFile] = rawSpec
	}

	return res
}

// GetSwagger returns the Swagger specification corresponding to the generated code
// in this file. The external references of Swagger specification are resolved.
// The logic of resolving external references is tightly connected to "import-mapping" feature.
// Externally referenced files must be embedded in the corresponding golang packages.
// Urls can be supported but this task was out of the scope.
func GetSwagger() (swagger *openapi3.T, err error) {
	resolvePath := PathToRawSpec("")

	loader := openapi3.NewLoader()
	loader.IsExternalRefsAllowed = true
	loader.ReadFromURIFunc = func(loader *openapi3.Loader, url *url.URL) ([]byte, error) {
		pathToFile := url.String()
		pathToFile = path.Clean(pathToFile)
		getSpec, ok := resolvePath[pathToFile]
		if !ok {
			err1 := fmt.Errorf("path not found: %s", pathToFile)
			return nil, err1
		}
		return getSpec()
	}
	var specData []byte
	specData, err = rawSpec()
	if err != nil {
		return
	}
	swagger, err = loader.LoadFromData(specData)
	if err != nil {
		return
	}
	return
}
