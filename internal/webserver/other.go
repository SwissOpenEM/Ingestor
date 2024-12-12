package webserver

import "context"

func (i *IngestorWebServerImplemenation) OtherControllerGetVersion(ctx context.Context, request OtherControllerGetVersionRequestObject) (OtherControllerGetVersionResponseObject, error) {
	return OtherControllerGetVersion200JSONResponse{
		Version: &i.version,
	}, nil
}

func (i *IngestorWebServerImplemenation) OtherControllerGetHealth(ctx context.Context, request OtherControllerGetHealthRequestObject) (OtherControllerGetHealthResponseObject, error) {
	return nil, nil
}
