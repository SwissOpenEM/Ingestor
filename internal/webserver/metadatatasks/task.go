package metadatatasks

import (
	"context"
	"errors"

	"github.com/SwissOpenEM/Ingestor/internal/metadataextractor"
)

type ExtractionProgress struct {
	extractorOutput string
	extractorError  error
	taskStdOut      string
	taskStdErr      string
	finished        bool
}

type task struct {
	ctx          context.Context
	datasetPath  string
	method       string
	taskProgress chan *ExtractionProgress
	taskFinish   chan bool
}

func (t *ExtractionProgress) setExtractorOutputAndErr(out string, err error) {
	if !t.finished {
		t.extractorOutput = out
		t.extractorError = err
		t.finished = true
	}
}

func (t *ExtractionProgress) GetExtractorOutput() string {
	return t.extractorOutput
}

func (t *ExtractionProgress) GetExtractorError() error {
	return t.extractorError
}

func (t *ExtractionProgress) setStdOut(output string) {
	if !t.finished {
		t.taskStdOut = output
	}
}

func (t *ExtractionProgress) setStdErr(output string) {
	if !t.finished {
		t.taskStdErr = output
	}
}

func (t *ExtractionProgress) GetStdOut() string {
	return t.taskStdOut
}

func (t *ExtractionProgress) GetStdErr() string {
	return t.taskStdErr
}

func NewTask(handler *metadataextractor.ExtractorHandler, ctx context.Context, datasetPath string, method string, taskProgress chan *ExtractionProgress, taskFinish chan bool) error {
	if datasetPath == "" {
		return errors.New("datasetPath can't be empty")
	}
	a := task{
		ctx:          ctx,
		datasetPath:  datasetPath,
		method:       method,
		taskProgress: taskProgress,
		taskFinish:   taskFinish,
	}
	_ = a
	return nil
}
