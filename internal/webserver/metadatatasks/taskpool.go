package metadatatasks

import (
	"context"
	"fmt"
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

func (p *MetadataExtractionTaskPool) NewTask(ctx context.Context, datasetPath string, method string) *ExtractionProgress {
	epc := ExtractionProgress{
		ProgressSignal: make(chan bool),
	}
	epc.setStdOut(fmt.Sprintf("waiting for a free worker, your number in the queue: %d", len(p.tasks)))
	p.tasks <- task{
		ctx:          ctx,
		datasetPath:  datasetPath,
		method:       method,
		taskProgress: &epc,
	}
	return &epc
}

func NewTaskPool(numWorkers uint, handler *metadataextractor.ExtractorHandler) *MetadataExtractionTaskPool {
	pool := MetadataExtractionTaskPool{
		tasks:   make(chan task, numWorkers),
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
		task.taskProgress = &ExtractionProgress{}
		out, err := pool.handler.ExtractMetadata(task.ctx, task.method, task.datasetPath, outputFolder, task.taskProgress.setStdOut, task.taskProgress.setStdErr)
		task.taskProgress.setExtractorOutputAndErr(out, err)
	}
}
