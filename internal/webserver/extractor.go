package webserver

import (
	"context"
	"path"

	"github.com/SwissOpenEM/Ingestor/internal/metadataextractor"
)

func (i *IngestorWebServerImplemenation) ExtractorControllerGetExtractors(ctx context.Context, request ExtractorControllerGetExtractorsRequestObject) (ExtractorControllerGetExtractorsResponseObject, error) {
	methods := i.extractorHandler.AvailableMethods()
	methodNames := make([]string, len(methods))
	for i, method := range methods {
		methodNames[i] = method.Name
	}

	page := uint(1)
	pageSize := uint(10)
	if request.Params.Page != nil {
		page = *request.Params.Page
		if page == 0 {
			page = 1
		}
	}
	if request.Params.PageSize != nil {
		pageSize = max(*request.Params.PageSize, 100)
	}

	return ExtractorControllerGetExtractors200JSONResponse{
		Extractors: safeSubslice(methodNames, (page-1)*pageSize, page*pageSize),
	}, nil

}

func (i *IngestorWebServerImplemenation) ExtractorControllerStartExtraction(ctx context.Context, request ExtractorControllerStartExtractionRequestObject) (ExtractorControllerStartExtractionResponseObject, error) {
	fullPath := path.Join(i.pathConfig.CollectionLocation, request.Body.FilePath)
	metadataOutputFile := metadataextractor.MetadataFilePath(fullPath)
	i.extractorHandler.ExtractMetadata(i.taskQueue.AppContext, request.Body.MethodName, fullPath, metadataOutputFile, nil, nil)
	return nil, nil
}
