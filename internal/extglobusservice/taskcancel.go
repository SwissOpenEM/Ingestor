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
		return newRequestError(400, parsedResp.JSON400.Message, parsedResp.JSON400.Details)
	case 401:
		return newRequestError(401, parsedResp.JSON401.Message, parsedResp.JSON401.Details)
	case 403:
		return newRequestError(403, parsedResp.JSON403.Message, parsedResp.JSON403.Details)
	case 500:
		return newRequestError(500, parsedResp.JSON500.Message, parsedResp.JSON500.Details)
	default:
		return fmt.Errorf("unknown status code: %d", parsedResp.StatusCode())
	}
}
