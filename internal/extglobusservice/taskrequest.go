//go:generate wget https://raw.githubusercontent.com/SwissOpenEM/globus-transfer-service/refs/heads/master/internal/api/openapi.yaml -O openapi.yaml
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

func newRequestError(code uint, message string, details string) error {
	return &RequestError{
		code:    code,
		message: message,
		details: details,
	}
}

func RequestExternalTransferTask(ctx context.Context, serviceUrl string, scicatToken string, srcFacility string, dstFacility string, scicatPid string, fileList *[]FileToTransfer) (string, error) {
	client, err := NewClient(serviceUrl)
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
			SourceFacility: srcFacility,
			DestFacility:   dstFacility,
			ScicatPid:      scicatPid,
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
		return *parsedResp.JSON200.JobId, nil
	case 400:
		return "", newRequestError(400, *parsedResp.JSON400.Message, *parsedResp.JSON400.Details)
	case 401:
		return "", newRequestError(401, *parsedResp.JSON401.Message, *parsedResp.JSON401.Details)
	case 403:
		return "", newRequestError(403, *parsedResp.JSON403.Message, *parsedResp.JSON403.Details)
	case 500:
		return "", newRequestError(500, *parsedResp.JSON500.Message, *parsedResp.JSON500.Details)
	case 503:
		return "", newRequestError(503, *parsedResp.JSON503.Message, *parsedResp.JSON503.Details)
	}
	return "", fmt.Errorf("unexpected status code: %d", parsedResp.StatusCode())
}
