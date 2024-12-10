package webserver

import (
	"context"
	"fmt"
	"path"

	"github.com/SwissOpenEM/Ingestor/internal/metadataextractor"
)

func (i *IngestorWebServerImplemenation) ExtractorControllerGetExtractorMethods(ctx context.Context, request ExtractorControllerGetExtractorMethodsRequestObject) (ExtractorControllerGetExtractorMethodsResponseObject, error) {
	// get methods
	methods := i.extractorHandler.AvailableMethods()

	// get indices
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

	// get subslice and convert to "DTO"
	methods = safeSubslice(methods, (page-1)*pageSize, page*pageSize)
	methodsDTO := make([]MethodItem, len(methods))
	for i, method := range methods {
		methodsDTO[i] = MethodItem(method)
	}

	// return result
	return ExtractorControllerGetExtractorMethods200JSONResponse{
		Methods: methodsDTO,
		Total:   len(methods),
	}, nil
}

func (i *IngestorWebServerImplemenation) ExtractorControllerStartExtraction(ctx context.Context, request ExtractorControllerStartExtractionRequestObject) (ExtractorControllerStartExtractionResponseObject, error) {
	// append collection path to input and generate extractor output filepath
	fullPath := path.Join(i.pathConfig.CollectionLocation, request.Body.FilePath)
	metadataOutputFile := metadataextractor.MetadataFilePath(fullPath)

	// stdout and stderr callbacks
	var cmdStdOut string
	stdOutCallback := func(out string) {
		cmdStdOut = out
	}
	var cmdStdErr string
	stdErrCallback := func(err string) {
		cmdStdErr = err
	}

	// extract metadata
	result, err := i.extractorHandler.ExtractMetadata(i.taskQueue.AppContext, request.Body.MethodName, fullPath, metadataOutputFile, stdOutCallback, stdErrCallback)
	if err != nil {
		if _, ok := err.(metadataextractor.ExtractionRequestError); ok {
			return ExtractorControllerStartExtraction400TextResponse(fmt.Sprintf("Metadata Extractor - invalid parameters error: %s", err.Error())), nil
		} else {
			return ExtractorControllerStartExtraction500TextResponse(fmt.Sprintf("Metadata Extractor - other error: %s", err.Error())), nil
		}
	}

	// return result
	return ExtractorControllerStartExtraction200JSONResponse{
		Result:    result,
		CmdStdOut: cmdStdOut,
		CmdStdErr: cmdStdErr,
	}, err
}
