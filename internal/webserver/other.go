package webserver

import (
	"context"

	"github.com/SwissOpenEM/Ingestor/internal/core"
)

func (i *IngestorWebServerImplemenation) OtherControllerGetVersion(ctx context.Context, request OtherControllerGetVersionRequestObject) (OtherControllerGetVersionResponseObject, error) {
	return OtherControllerGetVersion200JSONResponse{
		Version: &i.version,
	}, nil
}

func (i *IngestorWebServerImplemenation) OtherControllerGetHealth(ctx context.Context, request OtherControllerGetHealthRequestObject) (OtherControllerGetHealthResponseObject, error) {
	errors := map[string]string{}

	err := core.ScicatHealthTest(i.taskQueue.Config.Scicat.Host)
	if err != nil {
		errors["scicat"] = err.Error()
	}

	/*err = core.GlobusHealthCheck()
	if err != nil {
		errors["globus"] = err.Error()
	}*/

	if len(errors) > 0 {
		return OtherControllerGetHealth200JSONResponse{
			Status: "error",
			Errors: &errors,
		}, nil
	} else {
		return OtherControllerGetHealth200JSONResponse{
			Status: "ok",
		}, nil
	}
}
