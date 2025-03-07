package metadatatasks

import (
	"context"
	"sync"

	"github.com/SwissOpenEM/Ingestor/internal/metadataextractor"
	"github.com/alitto/pond/v2"
)

type MetadataExtractionTaskPool struct {
	pool              pond.Pool
	waitGroup         sync.WaitGroup
	extractionHandler *metadataextractor.ExtractorHandler
}

func (p *MetadataExtractionTaskPool) GetAvailableMethods() []metadataextractor.MethodAndSchema {
	return p.extractionHandler.AvailableMethods()
}

func (p *MetadataExtractionTaskPool) NewTask(ctx context.Context, datasetPath string, method string) (*ExtractionProgress, error) {
	progress := ExtractionProgress{
		ProgressSignal: make(chan bool, 1),
	}

	executeTask := func() {
		progress.setProgress()
		outputFile := metadataextractor.MetadataFilePath(datasetPath)
		out, err := p.extractionHandler.ExtractMetadata(ctx, method, datasetPath, outputFile, progress.setStdOut, progress.setStdErr)
		progress.setExtractorOutputAndErr(out, err)
	}

	p.pool.Submit(executeTask)
	return &progress, nil
}

func (p *MetadataExtractionTaskPool) GetHandler() *metadataextractor.ExtractorHandler {
	return p.extractionHandler
}

func NewTaskPool(queueSize int, maxConcurrency int, handler *metadataextractor.ExtractorHandler) *MetadataExtractionTaskPool {
	pondPool := pond.NewPool(int(maxConcurrency), pond.WithQueueSize(int(queueSize)))
	return NewTaskPoolFromPool(handler, pondPool)
}

func NewTaskPoolFromPool(handler *metadataextractor.ExtractorHandler, pool pond.Pool) *MetadataExtractionTaskPool {
	taskPool := MetadataExtractionTaskPool{
		pool:              pool,
		waitGroup:         sync.WaitGroup{},
		extractionHandler: handler,
	}
	return &taskPool
}
