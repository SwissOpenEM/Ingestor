package extglobusservice

import (
	"context"
	"fmt"
)

func CancelTask(ctx context.Context, serviceUrl string, scicatToken string, jobId string, deleteEntry bool) error {
	client, err := NewClient(serviceUrl)
	if err != nil {
		return err
	}

	rawResp, err := client.DeleteTransferTask(ctx, jobId, &DeleteTransferTaskParams{&deleteEntry})
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
