// Package extglobusservice provides primitives to interact with the openapi HTTP API.
//
// Code generated by github.com/oapi-codegen/oapi-codegen/v2 version v2.4.1 DO NOT EDIT.
package extglobusservice

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/oapi-codegen/runtime"
)

const (
	ScicatKeyAuthScopes = "ScicatKeyAuth.Scopes"
)

// FileToTransfer the file to transfer as part of a transfer request
type FileToTransfer struct {
	// IsSymlink specifies whether this file is a symlink
	IsSymlink bool `json:"isSymlink"`

	// Path the path of the file, it has to be relative to the dataset source folder
	Path string `json:"path"`
}

// GeneralErrorResponse defines model for GeneralErrorResponse.
type GeneralErrorResponse struct {
	// Details further details, debugging information
	Details *string `json:"details,omitempty"`

	// Message the error message
	Message *string `json:"message,omitempty"`
}

// PostTransferTaskJSONBody defines parameters for PostTransferTask.
type PostTransferTaskJSONBody struct {
	FileList *[]FileToTransfer `json:"fileList,omitempty"`
}

// PostTransferTaskParams defines parameters for PostTransferTask.
type PostTransferTaskParams struct {
	// SourceFacility the identifier name of the source facility
	SourceFacility string `form:"sourceFacility" json:"sourceFacility"`

	// DestFacility the path in the destination collection to use for the transfer
	DestFacility string `form:"destFacility" json:"destFacility"`

	// ScicatPid the pid of the dataset being transferred
	ScicatPid string `form:"scicatPid" json:"scicatPid"`
}

// DeleteTransferTaskParams defines parameters for DeleteTransferTask.
type DeleteTransferTaskParams struct {
	// Delete Enables/disables deleting from scicat job system. By default, it's disabled (false).
	Delete *bool `form:"delete,omitempty" json:"delete,omitempty"`
}

// PostTransferTaskJSONRequestBody defines body for PostTransferTask for application/json ContentType.
type PostTransferTaskJSONRequestBody PostTransferTaskJSONBody

// RequestEditorFn  is the function signature for the RequestEditor callback function
type RequestEditorFn func(ctx context.Context, req *http.Request) error

// Doer performs HTTP requests.
//
// The standard http.Client implements this interface.
type HttpRequestDoer interface {
	Do(req *http.Request) (*http.Response, error)
}

// Client which conforms to the OpenAPI3 specification for this service.
type Client struct {
	// The endpoint of the server conforming to this interface, with scheme,
	// https://api.deepmap.com for example. This can contain a path relative
	// to the server, such as https://api.deepmap.com/dev-test, and all the
	// paths in the swagger spec will be appended to the server.
	Server string

	// Doer for performing requests, typically a *http.Client with any
	// customized settings, such as certificate chains.
	Client HttpRequestDoer

	// A list of callbacks for modifying requests which are generated before sending over
	// the network.
	RequestEditors []RequestEditorFn
}

// ClientOption allows setting custom parameters during construction
type ClientOption func(*Client) error

// Creates a new Client, with reasonable defaults
func NewClient(server string, opts ...ClientOption) (*Client, error) {
	// create a client with sane default values
	client := Client{
		Server: server,
	}
	// mutate client and add all optional params
	for _, o := range opts {
		if err := o(&client); err != nil {
			return nil, err
		}
	}
	// ensure the server URL always has a trailing slash
	if !strings.HasSuffix(client.Server, "/") {
		client.Server += "/"
	}
	// create httpClient, if not already present
	if client.Client == nil {
		client.Client = &http.Client{}
	}
	return &client, nil
}

// WithHTTPClient allows overriding the default Doer, which is
// automatically created using http.Client. This is useful for tests.
func WithHTTPClient(doer HttpRequestDoer) ClientOption {
	return func(c *Client) error {
		c.Client = doer
		return nil
	}
}

// WithRequestEditorFn allows setting up a callback function, which will be
// called right before sending the request. This can be used to mutate the request.
func WithRequestEditorFn(fn RequestEditorFn) ClientOption {
	return func(c *Client) error {
		c.RequestEditors = append(c.RequestEditors, fn)
		return nil
	}
}

// The interface specification for the client above.
type ClientInterface interface {
	// PostTransferTaskWithBody request with any body
	PostTransferTaskWithBody(ctx context.Context, params *PostTransferTaskParams, contentType string, body io.Reader, reqEditors ...RequestEditorFn) (*http.Response, error)

	PostTransferTask(ctx context.Context, params *PostTransferTaskParams, body PostTransferTaskJSONRequestBody, reqEditors ...RequestEditorFn) (*http.Response, error)

	// DeleteTransferTask request
	DeleteTransferTask(ctx context.Context, scicatJobId string, params *DeleteTransferTaskParams, reqEditors ...RequestEditorFn) (*http.Response, error)
}

func (c *Client) PostTransferTaskWithBody(ctx context.Context, params *PostTransferTaskParams, contentType string, body io.Reader, reqEditors ...RequestEditorFn) (*http.Response, error) {
	req, err := NewPostTransferTaskRequestWithBody(c.Server, params, contentType, body)
	if err != nil {
		return nil, err
	}
	req = req.WithContext(ctx)
	if err := c.applyEditors(ctx, req, reqEditors); err != nil {
		return nil, err
	}
	return c.Client.Do(req)
}

func (c *Client) PostTransferTask(ctx context.Context, params *PostTransferTaskParams, body PostTransferTaskJSONRequestBody, reqEditors ...RequestEditorFn) (*http.Response, error) {
	req, err := NewPostTransferTaskRequest(c.Server, params, body)
	if err != nil {
		return nil, err
	}
	req = req.WithContext(ctx)
	if err := c.applyEditors(ctx, req, reqEditors); err != nil {
		return nil, err
	}
	return c.Client.Do(req)
}

func (c *Client) DeleteTransferTask(ctx context.Context, scicatJobId string, params *DeleteTransferTaskParams, reqEditors ...RequestEditorFn) (*http.Response, error) {
	req, err := NewDeleteTransferTaskRequest(c.Server, scicatJobId, params)
	if err != nil {
		return nil, err
	}
	req = req.WithContext(ctx)
	if err := c.applyEditors(ctx, req, reqEditors); err != nil {
		return nil, err
	}
	return c.Client.Do(req)
}

// NewPostTransferTaskRequest calls the generic PostTransferTask builder with application/json body
func NewPostTransferTaskRequest(server string, params *PostTransferTaskParams, body PostTransferTaskJSONRequestBody) (*http.Request, error) {
	var bodyReader io.Reader
	buf, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}
	bodyReader = bytes.NewReader(buf)
	return NewPostTransferTaskRequestWithBody(server, params, "application/json", bodyReader)
}

// NewPostTransferTaskRequestWithBody generates requests for PostTransferTask with any type of body
func NewPostTransferTaskRequestWithBody(server string, params *PostTransferTaskParams, contentType string, body io.Reader) (*http.Request, error) {
	var err error

	serverURL, err := url.Parse(server)
	if err != nil {
		return nil, err
	}

	operationPath := fmt.Sprintf("/transfer")
	if operationPath[0] == '/' {
		operationPath = "." + operationPath
	}

	queryURL, err := serverURL.Parse(operationPath)
	if err != nil {
		return nil, err
	}

	if params != nil {
		queryValues := queryURL.Query()

		if queryFrag, err := runtime.StyleParamWithLocation("form", true, "sourceFacility", runtime.ParamLocationQuery, params.SourceFacility); err != nil {
			return nil, err
		} else if parsed, err := url.ParseQuery(queryFrag); err != nil {
			return nil, err
		} else {
			for k, v := range parsed {
				for _, v2 := range v {
					queryValues.Add(k, v2)
				}
			}
		}

		if queryFrag, err := runtime.StyleParamWithLocation("form", true, "destFacility", runtime.ParamLocationQuery, params.DestFacility); err != nil {
			return nil, err
		} else if parsed, err := url.ParseQuery(queryFrag); err != nil {
			return nil, err
		} else {
			for k, v := range parsed {
				for _, v2 := range v {
					queryValues.Add(k, v2)
				}
			}
		}

		if queryFrag, err := runtime.StyleParamWithLocation("form", true, "scicatPid", runtime.ParamLocationQuery, params.ScicatPid); err != nil {
			return nil, err
		} else if parsed, err := url.ParseQuery(queryFrag); err != nil {
			return nil, err
		} else {
			for k, v := range parsed {
				for _, v2 := range v {
					queryValues.Add(k, v2)
				}
			}
		}

		queryURL.RawQuery = queryValues.Encode()
	}

	req, err := http.NewRequest("POST", queryURL.String(), body)
	if err != nil {
		return nil, err
	}

	req.Header.Add("Content-Type", contentType)

	return req, nil
}

// NewDeleteTransferTaskRequest generates requests for DeleteTransferTask
func NewDeleteTransferTaskRequest(server string, scicatJobId string, params *DeleteTransferTaskParams) (*http.Request, error) {
	var err error

	var pathParam0 string

	pathParam0, err = runtime.StyleParamWithLocation("simple", false, "scicatJobId", runtime.ParamLocationPath, scicatJobId)
	if err != nil {
		return nil, err
	}

	serverURL, err := url.Parse(server)
	if err != nil {
		return nil, err
	}

	operationPath := fmt.Sprintf("/transfer/%s", pathParam0)
	if operationPath[0] == '/' {
		operationPath = "." + operationPath
	}

	queryURL, err := serverURL.Parse(operationPath)
	if err != nil {
		return nil, err
	}

	if params != nil {
		queryValues := queryURL.Query()

		if params.Delete != nil {

			if queryFrag, err := runtime.StyleParamWithLocation("form", true, "delete", runtime.ParamLocationQuery, *params.Delete); err != nil {
				return nil, err
			} else if parsed, err := url.ParseQuery(queryFrag); err != nil {
				return nil, err
			} else {
				for k, v := range parsed {
					for _, v2 := range v {
						queryValues.Add(k, v2)
					}
				}
			}

		}

		queryURL.RawQuery = queryValues.Encode()
	}

	req, err := http.NewRequest("DELETE", queryURL.String(), nil)
	if err != nil {
		return nil, err
	}

	return req, nil
}

func (c *Client) applyEditors(ctx context.Context, req *http.Request, additionalEditors []RequestEditorFn) error {
	for _, r := range c.RequestEditors {
		if err := r(ctx, req); err != nil {
			return err
		}
	}
	for _, r := range additionalEditors {
		if err := r(ctx, req); err != nil {
			return err
		}
	}
	return nil
}

// ClientWithResponses builds on ClientInterface to offer response payloads
type ClientWithResponses struct {
	ClientInterface
}

// NewClientWithResponses creates a new ClientWithResponses, which wraps
// Client with return type handling
func NewClientWithResponses(server string, opts ...ClientOption) (*ClientWithResponses, error) {
	client, err := NewClient(server, opts...)
	if err != nil {
		return nil, err
	}
	return &ClientWithResponses{client}, nil
}

// WithBaseURL overrides the baseURL.
func WithBaseURL(baseURL string) ClientOption {
	return func(c *Client) error {
		newBaseURL, err := url.Parse(baseURL)
		if err != nil {
			return err
		}
		c.Server = newBaseURL.String()
		return nil
	}
}

// ClientWithResponsesInterface is the interface specification for the client with responses above.
type ClientWithResponsesInterface interface {
	// PostTransferTaskWithBodyWithResponse request with any body
	PostTransferTaskWithBodyWithResponse(ctx context.Context, params *PostTransferTaskParams, contentType string, body io.Reader, reqEditors ...RequestEditorFn) (*PostTransferTaskResponse, error)

	PostTransferTaskWithResponse(ctx context.Context, params *PostTransferTaskParams, body PostTransferTaskJSONRequestBody, reqEditors ...RequestEditorFn) (*PostTransferTaskResponse, error)

	// DeleteTransferTaskWithResponse request
	DeleteTransferTaskWithResponse(ctx context.Context, scicatJobId string, params *DeleteTransferTaskParams, reqEditors ...RequestEditorFn) (*DeleteTransferTaskResponse, error)
}

type PostTransferTaskResponse struct {
	Body         []byte
	HTTPResponse *http.Response
	JSON200      *struct {
		// JobId the SciCat job id of the transfer job
		JobId string `json:"jobId"`
	}
	JSON400 *GeneralErrorResponse
	JSON401 *GeneralErrorResponse
	JSON403 *GeneralErrorResponse
	JSON500 *GeneralErrorResponse
	JSON503 *GeneralErrorResponse
}

// Status returns HTTPResponse.Status
func (r PostTransferTaskResponse) Status() string {
	if r.HTTPResponse != nil {
		return r.HTTPResponse.Status
	}
	return http.StatusText(0)
}

// StatusCode returns HTTPResponse.StatusCode
func (r PostTransferTaskResponse) StatusCode() int {
	if r.HTTPResponse != nil {
		return r.HTTPResponse.StatusCode
	}
	return 0
}

type DeleteTransferTaskResponse struct {
	Body         []byte
	HTTPResponse *http.Response
	JSON400      *GeneralErrorResponse
	JSON401      *GeneralErrorResponse
	JSON403      *GeneralErrorResponse
	JSON500      *GeneralErrorResponse
}

// Status returns HTTPResponse.Status
func (r DeleteTransferTaskResponse) Status() string {
	if r.HTTPResponse != nil {
		return r.HTTPResponse.Status
	}
	return http.StatusText(0)
}

// StatusCode returns HTTPResponse.StatusCode
func (r DeleteTransferTaskResponse) StatusCode() int {
	if r.HTTPResponse != nil {
		return r.HTTPResponse.StatusCode
	}
	return 0
}

// PostTransferTaskWithBodyWithResponse request with arbitrary body returning *PostTransferTaskResponse
func (c *ClientWithResponses) PostTransferTaskWithBodyWithResponse(ctx context.Context, params *PostTransferTaskParams, contentType string, body io.Reader, reqEditors ...RequestEditorFn) (*PostTransferTaskResponse, error) {
	rsp, err := c.PostTransferTaskWithBody(ctx, params, contentType, body, reqEditors...)
	if err != nil {
		return nil, err
	}
	return ParsePostTransferTaskResponse(rsp)
}

func (c *ClientWithResponses) PostTransferTaskWithResponse(ctx context.Context, params *PostTransferTaskParams, body PostTransferTaskJSONRequestBody, reqEditors ...RequestEditorFn) (*PostTransferTaskResponse, error) {
	rsp, err := c.PostTransferTask(ctx, params, body, reqEditors...)
	if err != nil {
		return nil, err
	}
	return ParsePostTransferTaskResponse(rsp)
}

// DeleteTransferTaskWithResponse request returning *DeleteTransferTaskResponse
func (c *ClientWithResponses) DeleteTransferTaskWithResponse(ctx context.Context, scicatJobId string, params *DeleteTransferTaskParams, reqEditors ...RequestEditorFn) (*DeleteTransferTaskResponse, error) {
	rsp, err := c.DeleteTransferTask(ctx, scicatJobId, params, reqEditors...)
	if err != nil {
		return nil, err
	}
	return ParseDeleteTransferTaskResponse(rsp)
}

// ParsePostTransferTaskResponse parses an HTTP response from a PostTransferTaskWithResponse call
func ParsePostTransferTaskResponse(rsp *http.Response) (*PostTransferTaskResponse, error) {
	bodyBytes, err := io.ReadAll(rsp.Body)
	defer func() { _ = rsp.Body.Close() }()
	if err != nil {
		return nil, err
	}

	response := &PostTransferTaskResponse{
		Body:         bodyBytes,
		HTTPResponse: rsp,
	}

	switch {
	case strings.Contains(rsp.Header.Get("Content-Type"), "json") && rsp.StatusCode == 200:
		var dest struct {
			// JobId the SciCat job id of the transfer job
			JobId string `json:"jobId"`
		}
		if err := json.Unmarshal(bodyBytes, &dest); err != nil {
			return nil, err
		}
		response.JSON200 = &dest

	case strings.Contains(rsp.Header.Get("Content-Type"), "json") && rsp.StatusCode == 400:
		var dest GeneralErrorResponse
		if err := json.Unmarshal(bodyBytes, &dest); err != nil {
			return nil, err
		}
		response.JSON400 = &dest

	case strings.Contains(rsp.Header.Get("Content-Type"), "json") && rsp.StatusCode == 401:
		var dest GeneralErrorResponse
		if err := json.Unmarshal(bodyBytes, &dest); err != nil {
			return nil, err
		}
		response.JSON401 = &dest

	case strings.Contains(rsp.Header.Get("Content-Type"), "json") && rsp.StatusCode == 403:
		var dest GeneralErrorResponse
		if err := json.Unmarshal(bodyBytes, &dest); err != nil {
			return nil, err
		}
		response.JSON403 = &dest

	case strings.Contains(rsp.Header.Get("Content-Type"), "json") && rsp.StatusCode == 500:
		var dest GeneralErrorResponse
		if err := json.Unmarshal(bodyBytes, &dest); err != nil {
			return nil, err
		}
		response.JSON500 = &dest

	case strings.Contains(rsp.Header.Get("Content-Type"), "json") && rsp.StatusCode == 503:
		var dest GeneralErrorResponse
		if err := json.Unmarshal(bodyBytes, &dest); err != nil {
			return nil, err
		}
		response.JSON503 = &dest

	}

	return response, nil
}

// ParseDeleteTransferTaskResponse parses an HTTP response from a DeleteTransferTaskWithResponse call
func ParseDeleteTransferTaskResponse(rsp *http.Response) (*DeleteTransferTaskResponse, error) {
	bodyBytes, err := io.ReadAll(rsp.Body)
	defer func() { _ = rsp.Body.Close() }()
	if err != nil {
		return nil, err
	}

	response := &DeleteTransferTaskResponse{
		Body:         bodyBytes,
		HTTPResponse: rsp,
	}

	switch {
	case strings.Contains(rsp.Header.Get("Content-Type"), "json") && rsp.StatusCode == 400:
		var dest GeneralErrorResponse
		if err := json.Unmarshal(bodyBytes, &dest); err != nil {
			return nil, err
		}
		response.JSON400 = &dest

	case strings.Contains(rsp.Header.Get("Content-Type"), "json") && rsp.StatusCode == 401:
		var dest GeneralErrorResponse
		if err := json.Unmarshal(bodyBytes, &dest); err != nil {
			return nil, err
		}
		response.JSON401 = &dest

	case strings.Contains(rsp.Header.Get("Content-Type"), "json") && rsp.StatusCode == 403:
		var dest GeneralErrorResponse
		if err := json.Unmarshal(bodyBytes, &dest); err != nil {
			return nil, err
		}
		response.JSON403 = &dest

	case strings.Contains(rsp.Header.Get("Content-Type"), "json") && rsp.StatusCode == 500:
		var dest GeneralErrorResponse
		if err := json.Unmarshal(bodyBytes, &dest); err != nil {
			return nil, err
		}
		response.JSON500 = &dest

	}

	return response, nil
}
