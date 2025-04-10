//go:generate wget https://raw.githubusercontent.com/SwissOpenEM/globus-transfer-service/refs/heads/master/internal/api/openapi.yaml -O openapi.yaml
//go:generate go run github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen --config=./oapi-codegen-conf.yaml ./openapi.yaml
package extglobusservice

import (
	"context"
	"errors"
	"fmt"
	"net/http"
)

type RequestError4xx struct {
	msg string
}

func (e *RequestError4xx) Error() string {
	return e.msg
}

type RequestError5xx struct {
	msg string
}

func (e *RequestError5xx) Error() string {
	return e.msg
}

func newError(errType string, msg string) error {
	switch errType {
	case "req4xx":
		return &RequestError4xx{msg: msg}
	case "req5xx":
		return &RequestError5xx{msg: msg}
	default:
		return errors.New(msg)
	}
}

func RequestExternalTransferTask(ctx context.Context, serviceUrl string, scicatToken string, srcFacility string, dstFacility string, scicatPid string, fileList *[]FileToTransfer) error {
	client, err := NewClient(serviceUrl)
	if err != nil {
		return newError("system", err.Error())
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
		return err
	}

	parsedResp, err := ParsePostTransferTaskResponse(rawResp)
	if err != nil {
		return err
	}

	switch parsedResp.StatusCode() {
	case 200:
		return nil
	case 400:
		return newError("req4xx", "Error: 400, Message: '"+*parsedResp.JSON400.Message+"', Details: '"+*parsedResp.JSON400.Message+"'")
	case 401:
		return newError("req4xx", "Error: 401, Message: '"+*parsedResp.JSON401.Message+"', Details: '"+*parsedResp.JSON401.Message+"'")
	case 403:
		return newError("req4xx", "Error: 403, Message: '"+*parsedResp.JSON403.Message+"', Details: '"+*parsedResp.JSON403.Message+"'")
	case 500:
		return newError("req5xx", "Error: 500, Message: '"+*parsedResp.JSON500.Message+"', Details: '"+*parsedResp.JSON500.Message+"'")
	case 503:
		return newError("req5xx", "Error: 503, Message: '"+*parsedResp.JSON503.Message+"', Details: '"+*parsedResp.JSON503.Message+"'")
	}
	return fmt.Errorf("unexpected status code: %d", parsedResp.StatusCode())
}
