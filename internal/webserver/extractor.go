package webserver

import (
	"context"
)

func (i *IngestorWebServerImplemenation) ExtractorControllerGetExtractorMethods(ctx context.Context, request ExtractorControllerGetExtractorMethodsRequestObject) (ExtractorControllerGetExtractorMethodsResponseObject, error) {
	// get methods
	methods := i.extractorHandler.AvailableMethods()
	total := len(methods)

	// get indices
	page := uint(1)
	pageSize := uint(10)
	if request.Params.Page != nil {
		page = max(*request.Params.Page, 1)
	}
	if request.Params.PageSize != nil {
		pageSize = min(*request.Params.PageSize, 100)
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
		Total:   total,
	}, nil
}
