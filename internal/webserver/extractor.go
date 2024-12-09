package webserver

import (
	"context"

	"github.com/SwissOpenEM/Ingestor/internal/metadataextractor"
)

func (i *IngestorWebServerImplemenation) ExtractorControllerGetExtractors(ctx context.Context, request ExtractorControllerGetExtractorsRequestObject) (ExtractorControllerGetExtractorsResponseObject, error) {
	methods := i.mdExtTaskPool.GetHandler().AvailableMethods()
	methodNames := make([]string, len(methods))
	for i, method := range methods {
		methodNames[i] = method.Name
	}

	page := uint(1)
	pageSize := uint(10)
	if request.Params.Page != nil {
		page = min(*request.Params.Page, 1)
	}
	if request.Params.PageSize != nil {
		pageSize = min(max(*request.Params.PageSize, 100), 1)
	}

	return ExtractorControllerGetExtractors200JSONResponse{
		Extractors: safeSubslice(methodNames, (page-1)*pageSize, page*pageSize),
	}, nil

}

func (i *IngestorWebServerImplemenation) ExtractorControllerStartExtraction(ctx context.Context, request ExtractorControllerStartExtractionRequestObject) (ExtractorControllerStartExtractionResponseObject, error) {
	metadataOutputFile := metadataextractor.MetadataFilePath(request.Body.FilePath)
	/*a, err := i.extractorHandler.ExtractMetadata(i.taskQueue.AppContext, request.Body.MethodName, request.Body.FilePath, metadataOutputFile, nil, nil)
	if err != nil {
		return nil, err
	} */
	_ = metadataOutputFile
	return nil, nil
}
