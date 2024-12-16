package metadatatasks

import (
	"context"
	"errors"
	"sync"

	"github.com/SwissOpenEM/Ingestor/internal/metadataextractor"
)

type MetadataExtractionTaskPool struct {
	tasks   chan task
	wg      sync.WaitGroup
	handler *metadataextractor.ExtractorHandler
}

func (p *MetadataExtractionTaskPool) GetAvailableMethods() []metadataextractor.MethodAndSchema {
	return p.handler.AvailableMethods()
}

func (p *MetadataExtractionTaskPool) NewTask(ctx context.Context, datasetPath string, method string) (*ExtractionProgress, error) {
	epc := ExtractionProgress{
		ProgressSignal: make(chan bool, 1),
	}

	select {
	case p.tasks <- task{
		ctx:          ctx,
		datasetPath:  datasetPath,
		method:       method,
		taskProgress: &epc,
	}:
		return &epc, nil
	default:
		return nil, errors.New("task queue is full")
	}

}

func NewTaskPool(queueSize uint, numWorkers uint, handler *metadataextractor.ExtractorHandler) *MetadataExtractionTaskPool {
	pool := MetadataExtractionTaskPool{
		tasks:   make(chan task, queueSize),
		wg:      sync.WaitGroup{},
		handler: handler,
	}

	for i := uint(0); i < numWorkers; i++ {
		go worker(&pool)
	}

	return &pool
}

func worker(pool *MetadataExtractionTaskPool) {
	for {
		task := <-pool.tasks
		outputFolder := metadataextractor.MetadataFilePath(task.datasetPath)
		out, err := pool.handler.ExtractMetadata(task.ctx, task.method, task.datasetPath, outputFolder, task.taskProgress.setStdOut, task.taskProgress.setStdErr)
		task.taskProgress.setExtractorOutputAndErr(out, err)
	}
}
