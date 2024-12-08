// Package webserver provides primitives to interact with the openapi HTTP API.
//
// Code generated by github.com/oapi-codegen/oapi-codegen/v2 version v2.3.0 DO NOT EDIT.
package webserver

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"path"
	"strings"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/gin-gonic/gin"
	"github.com/oapi-codegen/runtime"
	strictgin "github.com/oapi-codegen/runtime/strictmiddleware/gin"
)

const (
	OpenIDScopes = "OpenID.Scopes"
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

// GetCallbackParams defines parameters for GetCallback.
type GetCallbackParams struct {
	// Code For handling the authorization code received from the OIDC provider
	Code string `form:"code" json:"code"`

	// State parameter for CSRF protection
	State string `form:"state" json:"state"`
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
	// OIDC callback
	// (GET /callback)
	GetCallback(c *gin.Context, params GetCallbackParams)
	// Ingest a new dataset
	// (POST /dataset)
	DatasetControllerIngestDataset(c *gin.Context)
	// OIDC login
	// (GET /login)
	GetLogin(c *gin.Context)
	// end user session
	// (GET /logout)
	GetLogout(c *gin.Context)
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

// GetCallback operation middleware
func (siw *ServerInterfaceWrapper) GetCallback(c *gin.Context) {

	var err error

	// Parameter object where we will unmarshal all parameters from the context
	var params GetCallbackParams

	// ------------- Required query parameter "code" -------------

	if paramValue := c.Query("code"); paramValue != "" {

	} else {
		siw.ErrorHandler(c, fmt.Errorf("Query argument code is required, but not found"), http.StatusBadRequest)
		return
	}

	err = runtime.BindQueryParameter("form", true, true, "code", c.Request.URL.Query(), &params.Code)
	if err != nil {
		siw.ErrorHandler(c, fmt.Errorf("Invalid format for parameter code: %w", err), http.StatusBadRequest)
		return
	}

	// ------------- Required query parameter "state" -------------

	if paramValue := c.Query("state"); paramValue != "" {

	} else {
		siw.ErrorHandler(c, fmt.Errorf("Query argument state is required, but not found"), http.StatusBadRequest)
		return
	}

	err = runtime.BindQueryParameter("form", true, true, "state", c.Request.URL.Query(), &params.State)
	if err != nil {
		siw.ErrorHandler(c, fmt.Errorf("Invalid format for parameter state: %w", err), http.StatusBadRequest)
		return
	}

	for _, middleware := range siw.HandlerMiddlewares {
		middleware(c)
		if c.IsAborted() {
			return
		}
	}

	siw.Handler.GetCallback(c, params)
}

// DatasetControllerIngestDataset operation middleware
func (siw *ServerInterfaceWrapper) DatasetControllerIngestDataset(c *gin.Context) {

	c.Set(OpenIDScopes, []string{"ingestor_write"})

	for _, middleware := range siw.HandlerMiddlewares {
		middleware(c)
		if c.IsAborted() {
			return
		}
	}

	siw.Handler.DatasetControllerIngestDataset(c)
}

// GetLogin operation middleware
func (siw *ServerInterfaceWrapper) GetLogin(c *gin.Context) {

	for _, middleware := range siw.HandlerMiddlewares {
		middleware(c)
		if c.IsAborted() {
			return
		}
	}

	siw.Handler.GetLogin(c)
}

// GetLogout operation middleware
func (siw *ServerInterfaceWrapper) GetLogout(c *gin.Context) {

	for _, middleware := range siw.HandlerMiddlewares {
		middleware(c)
		if c.IsAborted() {
			return
		}
	}

	siw.Handler.GetLogout(c)
}

// TransferControllerDeleteTransfer operation middleware
func (siw *ServerInterfaceWrapper) TransferControllerDeleteTransfer(c *gin.Context) {

	c.Set(OpenIDScopes, []string{"ingestor_write"})

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

	c.Set(OpenIDScopes, []string{"ingestor_read"})

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

	router.GET(options.BaseURL+"/callback", wrapper.GetCallback)
	router.POST(options.BaseURL+"/dataset", wrapper.DatasetControllerIngestDataset)
	router.GET(options.BaseURL+"/login", wrapper.GetLogin)
	router.GET(options.BaseURL+"/logout", wrapper.GetLogout)
	router.DELETE(options.BaseURL+"/transfer", wrapper.TransferControllerDeleteTransfer)
	router.GET(options.BaseURL+"/transfer", wrapper.TransferControllerGetTransfer)
	router.GET(options.BaseURL+"/version", wrapper.OtherControllerGetVersion)
}

type GetCallbackRequestObject struct {
	Params GetCallbackParams
}

type GetCallbackResponseObject interface {
	VisitGetCallbackResponse(w http.ResponseWriter) error
}

type GetCallback302ResponseHeaders struct {
	Location string
}

type GetCallback302Response struct {
	Headers GetCallback302ResponseHeaders
}

func (response GetCallback302Response) VisitGetCallbackResponse(w http.ResponseWriter) error {
	w.Header().Set("location", fmt.Sprint(response.Headers.Location))
	w.WriteHeader(302)
	return nil
}

type GetCallback400TextResponse string

func (response GetCallback400TextResponse) VisitGetCallbackResponse(w http.ResponseWriter) error {
	w.Header().Set("Content-Type", "text/plain")
	w.WriteHeader(400)

	_, err := w.Write([]byte(response))
	return err
}

type GetCallback500TextResponse string

func (response GetCallback500TextResponse) VisitGetCallbackResponse(w http.ResponseWriter) error {
	w.Header().Set("Content-Type", "text/plain")
	w.WriteHeader(500)

	_, err := w.Write([]byte(response))
	return err
}

type DatasetControllerIngestDatasetRequestObject struct {
	Body *DatasetControllerIngestDatasetJSONRequestBody
}

type DatasetControllerIngestDatasetResponseObject interface {
	VisitDatasetControllerIngestDatasetResponse(w http.ResponseWriter) error
}

type DatasetControllerIngestDataset200JSONResponse IngestorUiPostDatasetResponse

func (response DatasetControllerIngestDataset200JSONResponse) VisitDatasetControllerIngestDatasetResponse(w http.ResponseWriter) error {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(200)

	return json.NewEncoder(w).Encode(response)
}

type DatasetControllerIngestDataset400TextResponse string

func (response DatasetControllerIngestDataset400TextResponse) VisitDatasetControllerIngestDatasetResponse(w http.ResponseWriter) error {
	w.Header().Set("Content-Type", "text/plain")
	w.WriteHeader(400)

	_, err := w.Write([]byte(response))
	return err
}

type GetLoginRequestObject struct {
}

type GetLoginResponseObject interface {
	VisitGetLoginResponse(w http.ResponseWriter) error
}

type GetLogin302ResponseHeaders struct {
	Location string
}

type GetLogin302Response struct {
	Headers GetLogin302ResponseHeaders
}

func (response GetLogin302Response) VisitGetLoginResponse(w http.ResponseWriter) error {
	w.Header().Set("location", fmt.Sprint(response.Headers.Location))
	w.WriteHeader(302)
	return nil
}

type GetLogoutRequestObject struct {
}

type GetLogoutResponseObject interface {
	VisitGetLogoutResponse(w http.ResponseWriter) error
}

type GetLogout302ResponseHeaders struct {
	Location string
}

type GetLogout302Response struct {
	Headers GetLogout302ResponseHeaders
}

func (response GetLogout302Response) VisitGetLogoutResponse(w http.ResponseWriter) error {
	w.Header().Set("location", fmt.Sprint(response.Headers.Location))
	w.WriteHeader(302)
	return nil
}

type GetLogout500TextResponse string

func (response GetLogout500TextResponse) VisitGetLogoutResponse(w http.ResponseWriter) error {
	w.Header().Set("Content-Type", "text/plain")
	w.WriteHeader(500)

	_, err := w.Write([]byte(response))
	return err
}

type TransferControllerDeleteTransferRequestObject struct {
	Body *TransferControllerDeleteTransferJSONRequestBody
}

type TransferControllerDeleteTransferResponseObject interface {
	VisitTransferControllerDeleteTransferResponse(w http.ResponseWriter) error
}

type TransferControllerDeleteTransfer200JSONResponse IngestorUiDeleteTransferResponse

func (response TransferControllerDeleteTransfer200JSONResponse) VisitTransferControllerDeleteTransferResponse(w http.ResponseWriter) error {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(200)

	return json.NewEncoder(w).Encode(response)
}

type TransferControllerDeleteTransfer400TextResponse string

func (response TransferControllerDeleteTransfer400TextResponse) VisitTransferControllerDeleteTransferResponse(w http.ResponseWriter) error {
	w.Header().Set("Content-Type", "text/plain")
	w.WriteHeader(400)

	_, err := w.Write([]byte(response))
	return err
}

type TransferControllerGetTransferRequestObject struct {
	Params TransferControllerGetTransferParams
}

type TransferControllerGetTransferResponseObject interface {
	VisitTransferControllerGetTransferResponse(w http.ResponseWriter) error
}

type TransferControllerGetTransfer200JSONResponse IngestorUiGetTransferResponse

func (response TransferControllerGetTransfer200JSONResponse) VisitTransferControllerGetTransferResponse(w http.ResponseWriter) error {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(200)

	return json.NewEncoder(w).Encode(response)
}

type TransferControllerGetTransfer400TextResponse string

func (response TransferControllerGetTransfer400TextResponse) VisitTransferControllerGetTransferResponse(w http.ResponseWriter) error {
	w.Header().Set("Content-Type", "text/plain")
	w.WriteHeader(400)

	_, err := w.Write([]byte(response))
	return err
}

type OtherControllerGetVersionRequestObject struct {
}

type OtherControllerGetVersionResponseObject interface {
	VisitOtherControllerGetVersionResponse(w http.ResponseWriter) error
}

type OtherControllerGetVersion200JSONResponse IngestorUiOtherVersionResponse

func (response OtherControllerGetVersion200JSONResponse) VisitOtherControllerGetVersionResponse(w http.ResponseWriter) error {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(200)

	return json.NewEncoder(w).Encode(response)
}

// StrictServerInterface represents all server handlers.
type StrictServerInterface interface {
	// OIDC callback
	// (GET /callback)
	GetCallback(ctx context.Context, request GetCallbackRequestObject) (GetCallbackResponseObject, error)
	// Ingest a new dataset
	// (POST /dataset)
	DatasetControllerIngestDataset(ctx context.Context, request DatasetControllerIngestDatasetRequestObject) (DatasetControllerIngestDatasetResponseObject, error)
	// OIDC login
	// (GET /login)
	GetLogin(ctx context.Context, request GetLoginRequestObject) (GetLoginResponseObject, error)
	// end user session
	// (GET /logout)
	GetLogout(ctx context.Context, request GetLogoutRequestObject) (GetLogoutResponseObject, error)
	// Cancel a data transfer
	// (DELETE /transfer)
	TransferControllerDeleteTransfer(ctx context.Context, request TransferControllerDeleteTransferRequestObject) (TransferControllerDeleteTransferResponseObject, error)
	// Get list of transfers. Optional use the transferId parameter to only get one item.
	// (GET /transfer)
	TransferControllerGetTransfer(ctx context.Context, request TransferControllerGetTransferRequestObject) (TransferControllerGetTransferResponseObject, error)
	// Get the used ingestor version
	// (GET /version)
	OtherControllerGetVersion(ctx context.Context, request OtherControllerGetVersionRequestObject) (OtherControllerGetVersionResponseObject, error)
}

type StrictHandlerFunc = strictgin.StrictGinHandlerFunc
type StrictMiddlewareFunc = strictgin.StrictGinMiddlewareFunc

func NewStrictHandler(ssi StrictServerInterface, middlewares []StrictMiddlewareFunc) ServerInterface {
	return &strictHandler{ssi: ssi, middlewares: middlewares}
}

type strictHandler struct {
	ssi         StrictServerInterface
	middlewares []StrictMiddlewareFunc
}

// GetCallback operation middleware
func (sh *strictHandler) GetCallback(ctx *gin.Context, params GetCallbackParams) {
	var request GetCallbackRequestObject

	request.Params = params

	handler := func(ctx *gin.Context, request interface{}) (interface{}, error) {
		return sh.ssi.GetCallback(ctx, request.(GetCallbackRequestObject))
	}
	for _, middleware := range sh.middlewares {
		handler = middleware(handler, "GetCallback")
	}

	response, err := handler(ctx, request)

	if err != nil {
		ctx.Error(err)
		ctx.Status(http.StatusInternalServerError)
	} else if validResponse, ok := response.(GetCallbackResponseObject); ok {
		if err := validResponse.VisitGetCallbackResponse(ctx.Writer); err != nil {
			ctx.Error(err)
		}
	} else if response != nil {
		ctx.Error(fmt.Errorf("unexpected response type: %T", response))
	}
}

// DatasetControllerIngestDataset operation middleware
func (sh *strictHandler) DatasetControllerIngestDataset(ctx *gin.Context) {
	var request DatasetControllerIngestDatasetRequestObject

	var body DatasetControllerIngestDatasetJSONRequestBody
	if err := ctx.ShouldBindJSON(&body); err != nil {
		ctx.Status(http.StatusBadRequest)
		ctx.Error(err)
		return
	}
	request.Body = &body

	handler := func(ctx *gin.Context, request interface{}) (interface{}, error) {
		return sh.ssi.DatasetControllerIngestDataset(ctx, request.(DatasetControllerIngestDatasetRequestObject))
	}
	for _, middleware := range sh.middlewares {
		handler = middleware(handler, "DatasetControllerIngestDataset")
	}

	response, err := handler(ctx, request)

	if err != nil {
		ctx.Error(err)
		ctx.Status(http.StatusInternalServerError)
	} else if validResponse, ok := response.(DatasetControllerIngestDatasetResponseObject); ok {
		if err := validResponse.VisitDatasetControllerIngestDatasetResponse(ctx.Writer); err != nil {
			ctx.Error(err)
		}
	} else if response != nil {
		ctx.Error(fmt.Errorf("unexpected response type: %T", response))
	}
}

// GetLogin operation middleware
func (sh *strictHandler) GetLogin(ctx *gin.Context) {
	var request GetLoginRequestObject

	handler := func(ctx *gin.Context, request interface{}) (interface{}, error) {
		return sh.ssi.GetLogin(ctx, request.(GetLoginRequestObject))
	}
	for _, middleware := range sh.middlewares {
		handler = middleware(handler, "GetLogin")
	}

	response, err := handler(ctx, request)

	if err != nil {
		ctx.Error(err)
		ctx.Status(http.StatusInternalServerError)
	} else if validResponse, ok := response.(GetLoginResponseObject); ok {
		if err := validResponse.VisitGetLoginResponse(ctx.Writer); err != nil {
			ctx.Error(err)
		}
	} else if response != nil {
		ctx.Error(fmt.Errorf("unexpected response type: %T", response))
	}
}

// GetLogout operation middleware
func (sh *strictHandler) GetLogout(ctx *gin.Context) {
	var request GetLogoutRequestObject

	handler := func(ctx *gin.Context, request interface{}) (interface{}, error) {
		return sh.ssi.GetLogout(ctx, request.(GetLogoutRequestObject))
	}
	for _, middleware := range sh.middlewares {
		handler = middleware(handler, "GetLogout")
	}

	response, err := handler(ctx, request)

	if err != nil {
		ctx.Error(err)
		ctx.Status(http.StatusInternalServerError)
	} else if validResponse, ok := response.(GetLogoutResponseObject); ok {
		if err := validResponse.VisitGetLogoutResponse(ctx.Writer); err != nil {
			ctx.Error(err)
		}
	} else if response != nil {
		ctx.Error(fmt.Errorf("unexpected response type: %T", response))
	}
}

// TransferControllerDeleteTransfer operation middleware
func (sh *strictHandler) TransferControllerDeleteTransfer(ctx *gin.Context) {
	var request TransferControllerDeleteTransferRequestObject

	var body TransferControllerDeleteTransferJSONRequestBody
	if err := ctx.ShouldBindJSON(&body); err != nil {
		ctx.Status(http.StatusBadRequest)
		ctx.Error(err)
		return
	}
	request.Body = &body

	handler := func(ctx *gin.Context, request interface{}) (interface{}, error) {
		return sh.ssi.TransferControllerDeleteTransfer(ctx, request.(TransferControllerDeleteTransferRequestObject))
	}
	for _, middleware := range sh.middlewares {
		handler = middleware(handler, "TransferControllerDeleteTransfer")
	}

	response, err := handler(ctx, request)

	if err != nil {
		ctx.Error(err)
		ctx.Status(http.StatusInternalServerError)
	} else if validResponse, ok := response.(TransferControllerDeleteTransferResponseObject); ok {
		if err := validResponse.VisitTransferControllerDeleteTransferResponse(ctx.Writer); err != nil {
			ctx.Error(err)
		}
	} else if response != nil {
		ctx.Error(fmt.Errorf("unexpected response type: %T", response))
	}
}

// TransferControllerGetTransfer operation middleware
func (sh *strictHandler) TransferControllerGetTransfer(ctx *gin.Context, params TransferControllerGetTransferParams) {
	var request TransferControllerGetTransferRequestObject

	request.Params = params

	handler := func(ctx *gin.Context, request interface{}) (interface{}, error) {
		return sh.ssi.TransferControllerGetTransfer(ctx, request.(TransferControllerGetTransferRequestObject))
	}
	for _, middleware := range sh.middlewares {
		handler = middleware(handler, "TransferControllerGetTransfer")
	}

	response, err := handler(ctx, request)

	if err != nil {
		ctx.Error(err)
		ctx.Status(http.StatusInternalServerError)
	} else if validResponse, ok := response.(TransferControllerGetTransferResponseObject); ok {
		if err := validResponse.VisitTransferControllerGetTransferResponse(ctx.Writer); err != nil {
			ctx.Error(err)
		}
	} else if response != nil {
		ctx.Error(fmt.Errorf("unexpected response type: %T", response))
	}
}

// OtherControllerGetVersion operation middleware
func (sh *strictHandler) OtherControllerGetVersion(ctx *gin.Context) {
	var request OtherControllerGetVersionRequestObject

	handler := func(ctx *gin.Context, request interface{}) (interface{}, error) {
		return sh.ssi.OtherControllerGetVersion(ctx, request.(OtherControllerGetVersionRequestObject))
	}
	for _, middleware := range sh.middlewares {
		handler = middleware(handler, "OtherControllerGetVersion")
	}

	response, err := handler(ctx, request)

	if err != nil {
		ctx.Error(err)
		ctx.Status(http.StatusInternalServerError)
	} else if validResponse, ok := response.(OtherControllerGetVersionResponseObject); ok {
		if err := validResponse.VisitOtherControllerGetVersionResponse(ctx.Writer); err != nil {
			ctx.Error(err)
		}
	} else if response != nil {
		ctx.Error(fmt.Errorf("unexpected response type: %T", response))
	}
}

// Base64 encoded, gzipped, json marshaled Swagger object
var swaggerSpec = []string{

	"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",
	"becOi6bLjObqnG5n278DAAD//2A7C5WhFwAA",
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
