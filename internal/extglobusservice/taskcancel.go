package extglobusservice

import (
	"context"
	"fmt"
	"net/http"
)

func CancelTask(ctx context.Context, serviceURL string, scicatToken string, jobID string, deleteEntry bool) error {
	client, err := NewClient(serviceURL)
	if err != nil {
		return err
	}

	scicatKeyAuth := func(ctx context.Context, req *http.Request) error {
		req.Header.Set("SciCat-API-Key", scicatToken)
		return nil
	}

	rawResp, err := client.DeleteTransferTask(ctx, jobID, &DeleteTransferTaskParams{&deleteEntry}, scicatKeyAuth)
	if err != nil {
		return err
	}

	parsedResp, err := ParseDeleteTransferTaskResponse(rawResp)
	if err != nil {
		return err
	}

	switch parsedResp.StatusCode() {
	case 200:
		return nil
	case 400:
		return newRequestErrorFromResponse(400, parsedResp.JSON400)
	case 401:
		return newRequestErrorFromResponse(401, parsedResp.JSON401)
	case 403:
		return newRequestErrorFromResponse(403, parsedResp.JSON403)
	case 500:
		return newRequestErrorFromResponse(500, parsedResp.JSON500)
	default:
		return fmt.Errorf("unknown status code: %d", parsedResp.StatusCode())
	}
}
