package webserver

import "context"

// OtherControllerGetVersion implements ServerInterface.
//
// @Description     Get the used ingestor version
// @Tags            other
// @Produce         json
// @Success         200      {object} webserver.OtherControllerGetVersion200JSONResponse "returns the version of the servedrf"
// @Router			/version [get]
func (i *IngestorWebServerImplemenation) OtherControllerGetVersion(ctx context.Context, request OtherControllerGetVersionRequestObject) (OtherControllerGetVersionResponseObject, error) {
	return OtherControllerGetVersion200JSONResponse{
		Version: &i.version,
	}, nil
}

func (i *IngestorWebServerImplemenation) OtherControllerGetHealth(ctx context.Context, request OtherControllerGetHealthRequestObject) (OtherControllerGetHealthResponseObject, error) {
	return nil, nil
}
