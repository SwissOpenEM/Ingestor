package webserver

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"path"

	"github.com/SwissOpenEM/Ingestor/internal/webserver/metadatatasks"
)

type ResponseWriter struct {
	ctx        context.Context
	ep         *metadatatasks.ExtractionProgress
	cancelTask context.CancelFunc
}

func (r ResponseWriter) VisitExtractMetadataResponse(writer http.ResponseWriter) error {
	defer r.cancelTask()
	writer.Header().Add("Content-Type", "text/event-stream")
	writer.Header().Add("Cache-Control", "no-cache")
	writer.Header().Add("Connection", "keep-alive")
	for {
		select {
		case _, ok := <-r.ep.ProgressSignal:
			json, err := json.Marshal(progressToDto(r.ep))
			if err != nil {
				return err
			}
			writer.Write([]byte("data: " + base64.StdEncoding.EncodeToString(json)))
			if !ok {
				return nil
			}
		case <-r.ctx.Done():
			return nil
		}
	}
}

func (i *IngestorWebServerImplemenation) ExtractMetadata(ctx context.Context, request ExtractMetadataRequestObject) (ExtractMetadataResponseObject, error) {
	// append collection path to input and generate extractor output filepath
	fullPath := path.Join(i.pathConfig.CollectionLocation, request.Body.FilePath)

	// extract metadata
	cancelCtx, cancel := context.WithCancel(ctx)
	progress := i.metp.NewTask(cancelCtx, fullPath, request.Body.MethodName)

	return ResponseWriter{ctx: ctx, ep: progress, cancelTask: cancel}, nil
}

type progressDto struct {
	StdOut string  `json:"std_out"`
	StdErr string  `json:"std_err"`
	Result *string `json:"result,omitempty"`
	Err    *string `json:"err,omitempty"`
}

func progressToDto(p *metadatatasks.ExtractionProgress) progressDto {
	return progressDto{
		StdOut: p.GetStdOut(),
		StdErr: p.GetStdErr(),
		Result: getStrPointerOrNil(p.GetExtractorOutput()),
		Err:    getStrPointerOrNil(getErrMsgIfNotNil(p.GetExtractorError())),
	}
}

func getErrMsgIfNotNil(err error) string {
	errMsg := ""
	if err != nil {
		errMsg = err.Error()
	}
	return errMsg
}

func getStrPointerOrNil(value string) *string {
	if value == "" {
		return nil
	}
	return &value
}
