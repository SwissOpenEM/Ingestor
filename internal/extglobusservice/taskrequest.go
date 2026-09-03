//go:generate go run github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen --config=./oapi-codegen-conf.yaml ./openapi.yaml
package extglobusservice

import (
	"context"
	"fmt"
	"net/http"
)

type RequestError struct {
	code    uint
	message string
	details string
}

func (e *RequestError) Error() string {
	return e.message
}

func (e *RequestError) Details() string {
	return e.details
}

func (e *RequestError) Code() uint {
	return e.code
}

func newRequestError(code uint, message *string, details *string) error {
	retMsg := ""
	retDetails := ""
	if message != nil {
		retMsg = *message
	}
	if details != nil {
		retDetails = *details
	}
	return &RequestError{
		code:    code,
		message: retMsg,
		details: retDetails,
	}
}

// newRequestErrorFromResponse builds a RequestError from a GeneralErrorResponse,
// which is nil when the server returned a non-JSON body for the status code
// (e.g. an error page from a proxy in front of the service).
func newRequestErrorFromResponse(code uint, resp *GeneralErrorResponse) error {
	if resp == nil {
		return newRequestError(code, nil, nil)
	}
	return newRequestError(code, resp.Message, resp.Details)
}

func RequestExternalTransferTask(ctx context.Context, serviceURL string, scicatToken string, srcFacility string, dstFacility string, scicatPid string, autoArchive bool, collectionRootPath string, fileList *[]FileToTransfer) (string, error) {
	client, err := NewClient(serviceURL)
	if err != nil {
		return "", err
	}

	scicatKeyAuth := func(ctx context.Context, req *http.Request) error {
		req.Header.Set("SciCat-API-Key", scicatToken)
		return nil
	}

	rawResp, err := client.PostTransferTask(
		ctx,
		&PostTransferTaskParams{
			SourceFacility:     srcFacility,
			DestFacility:       dstFacility,
			ScicatPid:          scicatPid,
			CollectionRootPath: collectionRootPath,
			AutoArchive:        &autoArchive,
		},
		PostTransferTaskJSONRequestBody{
			FileList: fileList,
		},
		scicatKeyAuth,
	)
	if err != nil {
		return "", err
	}

	parsedResp, err := ParsePostTransferTaskResponse(rawResp)
	if err != nil {
		return "", err
	}

	switch parsedResp.StatusCode() {
	case 200:
		if parsedResp.JSON200 == nil {
			return "", fmt.Errorf("external globus task request error - unexpected content type for status 200")
		}
		return parsedResp.JSON200.JobId, nil
	case 400:
		return "", newRequestErrorFromResponse(400, parsedResp.JSON400)
	case 401:
		return "", newRequestErrorFromResponse(401, parsedResp.JSON401)
	case 403:
		return "", newRequestErrorFromResponse(403, parsedResp.JSON403)
	case 500:
		return "", newRequestErrorFromResponse(500, parsedResp.JSON500)
	case 503:
		return "", newRequestErrorFromResponse(503, parsedResp.JSON503)
	}
	return "", fmt.Errorf("external globus task request error - unexpected status code: %d", parsedResp.StatusCode())
}
