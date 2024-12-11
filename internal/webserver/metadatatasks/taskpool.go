package metadatatasks

import (
	"sync"

	"github.com/SwissOpenEM/Ingestor/internal/metadataextractor"
)

type MetadataExtractionTaskPool struct {
	tasks   chan task
	wg      sync.WaitGroup
	handler *metadataextractor.ExtractorHandler
}

func (p *MetadataExtractionTaskPool) GetHandler() *metadataextractor.ExtractorHandler {
	return p.handler
}

func NewTaskPool(numWorkers uint) *MetadataExtractionTaskPool {
	pool := MetadataExtractionTaskPool{
		tasks: make(chan task, numWorkers),
		wg:    sync.WaitGroup{},
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
		progress := ExtractionProgress{}
		task.taskProgress <- &progress
		out, err := pool.handler.ExtractMetadata(task.ctx, task.method, task.datasetPath, outputFolder, progress.setStdOut, progress.setStdErr)
		progress.setExtractorOutputAndErr(out, err)
		task.taskFinish <- true
	}
}
