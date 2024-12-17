package metadatatasks

import (
	"context"
)

type ExtractionProgress struct {
	extractorOutput string
	extractorError  error
	taskStdOut      string
	taskStdErr      string
	finished        bool
	ProgressSignal  chan bool
}

type task struct {
	ctx          context.Context
	datasetPath  string
	method       string
	taskProgress *ExtractionProgress
}

func (t *ExtractionProgress) setExtractorOutputAndErr(out string, err error) {
	if !t.finished {
		t.extractorOutput = out
		t.extractorError = err
		t.finished = true
		close(t.ProgressSignal)
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
		t.setProgress()
	}
}

func (t *ExtractionProgress) setStdErr(output string) {
	if !t.finished {
		t.taskStdErr = output
		t.setProgress()
	}
}

func (t *ExtractionProgress) setProgress() {
	select {
	case t.ProgressSignal <- true:
	default:
	}
}

func (t *ExtractionProgress) GetStdOut() string {
	return t.taskStdOut
}

func (t *ExtractionProgress) GetStdErr() string {
	return t.taskStdErr
}
