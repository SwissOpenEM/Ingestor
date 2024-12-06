package metadatatasks

import (
	"context"
	"errors"

	"github.com/SwissOpenEM/Ingestor/internal/metadataextractor"
)

type ExtractionProgress struct {
	ExtractorOutput string
	ExtractorError  error
	TaskStdOut      string
	TaskStdErr      string
}

type task struct {
	handler      *metadataextractor.ExtractorHandler
	ctx          context.Context
	datasetPath  string
	method       string
	taskProgress chan ExtractionProgress
	taskFinish   chan bool
}

func (t *ExtractionProgress) stdOutCallback(output string) {
	t.TaskStdOut = output
}

func (t *ExtractionProgress) stdErrCallback(output string) {
	t.TaskStdErr = output
}

func NewTask(handler *metadataextractor.ExtractorHandler, ctx context.Context, datasetPath string, method string, taskProgress chan ExtractionProgress, taskFinish chan bool) error {
	if datasetPath == "" {
		return errors.New("datasetPath can't be empty")
	}
	a := task{
		handler:      handler,
		ctx:          ctx,
		datasetPath:  datasetPath,
		method:       method,
		taskProgress: taskProgress,
		taskFinish:   taskFinish,
	}
	_ = a
	return nil
}
