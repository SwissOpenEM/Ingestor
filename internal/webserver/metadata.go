package webserver

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"path"
	"time"

	"github.com/SwissOpenEM/Ingestor/internal/webserver/metadatatasks"
	"github.com/gin-gonic/gin"
)

type ResponseWriter struct {
	ctx                context.Context
	req                ExtractMetadataRequestObject
	metp               *metadatatasks.MetadataExtractionTaskPool
	collectionLocation string
}

func (r ResponseWriter) VisitExtractMetadataResponse(writer http.ResponseWriter) error {
	// kind of hackish, but only the pure gin way seems to work for SSE
	g := r.ctx.(*gin.Context)
	g.Writer.Header().Add("Content-Type", "text/event-stream")
	g.Writer.Header().Add("Cache-Control", "no-cache")
	g.Writer.Header().Add("Connection", "keep-alive")

	// append collection path to input and generate extractor output filepath
	fullPath := path.Join(r.collectionLocation, r.req.Body.FilePath)

	// extract metadata
	cancelCtx, cancel := context.WithCancel(r.ctx)
	defer cancel() // cancel ongoing job if client drops connection (TODO: test whether solution works)
	var progress *metadatatasks.ExtractionProgress
	var sleep, toQueue bool = false, true
	g.Stream(func(w io.Writer) bool {
		// queue task
		if toQueue {
			if sleep {
				time.Sleep(1 * time.Minute)
			}
			var err error
			progress, err = r.metp.NewTask(cancelCtx, fullPath, r.req.Body.MethodName)
			if err == nil {
				g.SSEvent("message", []byte("Your metadata extraction request is in the queue."))
				toQueue = false
				return true
			} else {
				g.SSEvent("message", []byte("task pool is full. Retrying in 1 minute..."))
				sleep = true
				return true
			}
		}

		// follow task progress
		select {
		case _, ok := <-progress.ProgressSignal:
			json, err := json.Marshal(progressToDto(progress))
			if err != nil {
				g.SSEvent("error", "couldn't marshal the progress json")
				return false
			}
			g.SSEvent("progress", json)
			g.Writer.Flush()
			if !ok {
				return false
			}
			return true
		case <-r.ctx.Done():
			return false // we get here if the client drops the connection
		}
	})
	return nil
}

func (i *IngestorWebServerImplemenation) ExtractMetadata(ctx context.Context, request ExtractMetadataRequestObject) (ExtractMetadataResponseObject, error) {
	return ResponseWriter{ctx: ctx, metp: i.metp, req: request, collectionLocation: i.pathConfig.CollectionLocation}, nil
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
