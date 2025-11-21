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

func RequestExternalTransferTask(ctx context.Context, serviceURL string, scicatToken string, srcFacility string, dstFacility string, scicatPid string, collectionRootPath string) (string, error) {
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
		return parsedResp.JSON200.JobId, nil
	case 400:
		return "", newRequestError(400, parsedResp.JSON400.Message, parsedResp.JSON400.Details)
	case 401:
		return "", newRequestError(401, parsedResp.JSON401.Message, parsedResp.JSON401.Details)
	case 403:
		return "", newRequestError(403, parsedResp.JSON403.Message, parsedResp.JSON403.Details)
	case 500:
		return "", newRequestError(500, parsedResp.JSON500.Message, parsedResp.JSON500.Details)
	case 503:
		return "", newRequestError(503, parsedResp.JSON503.Message, parsedResp.JSON503.Details)
	}
	return "", fmt.Errorf("external globus task request error - unexpected status code: %d", parsedResp.StatusCode())
}
