package metadatatasks

import (
	"sync"

	"github.com/SwissOpenEM/Ingestor/internal/metadataextractor"
)

type MetadataExtractionTaskPool struct {
	tasks chan task
	wg    sync.WaitGroup
}

func NewTaskPool(numWorkers int) *MetadataExtractionTaskPool {
	pool := MetadataExtractionTaskPool{
		tasks: make(chan task, numWorkers),
		wg:    sync.WaitGroup{},
	}

	for i := 0; i < numWorkers; i++ {
		go worker(&pool)
	}

	return &pool
}

func worker(pool *MetadataExtractionTaskPool) {
	for {
		task := <-pool.tasks
		outputFolder := metadataextractor.MetadataFilePath(task.datasetPath)
		progress := ExtractionProgress{}
		task.handler.ExtractMetadata(task.ctx, task.method, task.datasetPath, outputFolder)
	}
}
