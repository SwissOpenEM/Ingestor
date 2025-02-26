package metadatatasks

import (
	"context"
	"sync"

	"github.com/SwissOpenEM/Ingestor/internal/metadataextractor"
	"github.com/alitto/pond/v2"
)

type MetadataExtractionTaskPool struct {
	p  pond.Pool
	wg sync.WaitGroup
	h  *metadataextractor.ExtractorHandler
}

func (p *MetadataExtractionTaskPool) GetAvailableMethods() []metadataextractor.MethodAndSchema {
	return p.h.AvailableMethods()
}

func (p *MetadataExtractionTaskPool) NewTask(ctx context.Context, datasetPath string, method string) (*ExtractionProgress, error) {
	epc := ExtractionProgress{
		ProgressSignal: make(chan bool, 1),
	}

	executeTask := func() {
		epc.setProgress()
		outputFile := metadataextractor.MetadataFilePath(datasetPath)
		out, err := p.h.ExtractMetadata(ctx, method, datasetPath, outputFile, epc.setStdOut, epc.setStdErr)
		epc.setExtractorOutputAndErr(out, err)
	}

	p.p.Submit(executeTask)
	return &epc, nil
}

func NewTaskPool(queueSize int, maxConcurrency int, handler *metadataextractor.ExtractorHandler) *MetadataExtractionTaskPool {
	pondPool := pond.NewPool(int(maxConcurrency), pond.WithQueueSize(int(queueSize)))
	pool := MetadataExtractionTaskPool{
		p:  pondPool,
		wg: sync.WaitGroup{},
		h:  handler,
	}

	return &pool
}
