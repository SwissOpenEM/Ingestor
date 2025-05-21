package webserver

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"path/filepath"
	"time"

	b64 "encoding/base64"

	"github.com/SwissOpenEM/Ingestor/internal/datasetaccess"
	"github.com/SwissOpenEM/Ingestor/internal/webserver/collections"
	"github.com/SwissOpenEM/Ingestor/internal/webserver/metadatatasks"
	"github.com/gin-gonic/gin"
)

type ResponseWriter struct {
	ctx                context.Context
	req                ExtractMetadataRequestObject
	metadataTaskPool   *metadatatasks.MetadataExtractionTaskPool
	collectionLocation string
}

func (r ResponseWriter) VisitExtractMetadataResponse(writer http.ResponseWriter) error {
	// kind of hackish, but only the pure gin way seems to work for SSE
	g := r.ctx.(*gin.Context)
	g.Writer.Header().Add("Content-Type", "text/event-stream")
	g.Writer.Header().Add("Cache-Control", "no-cache")
	g.Writer.Header().Add("Connection", "keep-alive")

	// append collection path to input and generate extractor output filepath
	fullPath := filepath.Join(r.collectionLocation, filepath.Clean(r.req.Params.FilePath))

	// extract metadata
	cancelCtx, cancel := context.WithCancel(r.ctx)
	defer cancel() // cancel ongoing job if client drops connection (TODO: test whether solution works)
	var progress *metadatatasks.ExtractionProgress
	var sleep, queueing, waitForWorker bool = false, true, true
	var workerWaitingTimer <-chan time.Time
	g.Stream(func(w io.Writer) bool {
		// queue the task
		if queueing {
			if sleep {
				select {
				case <-time.After(1 * time.Minute):
				case <-g.Request.Context().Done():
					return false // client drops connection
				}
			}
			var err error
			progress, err = r.metadataTaskPool.NewTask(cancelCtx, fullPath, r.req.Params.MethodName)
			if err == nil {
				g.SSEvent("message", "Your metadata extraction request is in the queue.")
				queueing = false
				workerWaitingTimer = time.After(1 * time.Minute)
				return true
			} else {
				g.SSEvent("message", "Task pool is full. Retrying in 1 minute...")
				sleep = true
				return true
			}
		}

		// wait for worker
		if waitForWorker {
			select {
			case <-progress.ProgressSignal:
				g.SSEvent("message", "Extraction started.")
				waitForWorker = false
				select { // resetting progress signal to print out initial state in next block
				case progress.ProgressSignal <- true:
				default:
				}
			case <-workerWaitingTimer:
				g.SSEvent("message", "Still waiting for a free worker...`")
				workerWaitingTimer = time.After(1 * time.Minute)
				return true
			case <-g.Request.Context().Done():
				return false // client drops connection
			}
		}

		// follow task progress
		select {
		case _, ok := <-progress.ProgressSignal:
			json, err := json.Marshal(progressToDto(progress))
			if err != nil {
				g.SSEvent("error", "Couldn't marshal the progress json.")
				return false
			}
			b64json := b64.StdEncoding.EncodeToString([]byte(json))
			g.SSEvent("progress", b64json)
			g.Writer.Flush()
			if !ok {
				return false
			}
			return true
		case <-r.ctx.Done():
			return false // client drops connection
		}
	})
	return nil
}

func (i *IngestorWebServerImplemenation) ExtractMetadata(ctx context.Context, request ExtractMetadataRequestObject) (ExtractMetadataResponseObject, error) {
	// get collection location and relative path from input path
	_, colPath, relPath, err := collections.GetPathDetails(i.pathConfig.CollectionLocations, filepath.Clean(request.Params.FilePath))
	if err != nil {
		return ExtractMetadatadefaultJSONResponse{
			Body: Error{
				Code:    "400",
				Message: err.Error(),
			},
			StatusCode: 400}, nil
	}
	request.Params.FilePath = relPath

	// check if path is dir
	absPath := filepath.Join(colPath, relPath)
	err = datasetaccess.IsFolderCheck(absPath)
	if err != nil {
		return ExtractMetadatadefaultJSONResponse{
			Body: Error{
				Code:    "400",
				Message: err.Error(),
			},
			StatusCode: 400}, nil
	}

	// dataset access checks
	if !i.disableAuth {
		err = datasetaccess.CheckUserAccess(ctx, absPath)
		if _, ok := err.(*datasetaccess.AccessError); ok {
			return ExtractMetadatadefaultJSONResponse{
				Body: Error{
					Code:    "401",
					Message: "unauthorized: " + err.Error(),
				},
				StatusCode: 401}, nil
		} else if err != nil {
			slog.Error("user access error", "error", err.Error())
			return ExtractMetadatadefaultJSONResponse{
				Body: Error{
					Code:    "500",
					Message: "internal server error - user access error",
				},
				StatusCode: 500}, nil
		}
	}

	// start streaming the extraction process
	return ResponseWriter{ctx: ctx, metadataTaskPool: i.metp, req: request, collectionLocation: colPath}, nil
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
