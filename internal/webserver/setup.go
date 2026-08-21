package webserver

import (
	"context"
	"fmt"
	"log"
	"log/slog"
	"os"
	"strings"

	"github.com/SwissOpenEM/Ingestor/internal/core"
	"github.com/SwissOpenEM/Ingestor/internal/metadataextractor"
	"github.com/SwissOpenEM/Ingestor/internal/s3upload"
	"github.com/SwissOpenEM/Ingestor/internal/webserver/metadatatasks"
	"github.com/alitto/pond/v2"
	"github.com/oapi-codegen/runtime"
)

// Should run once at the start of the program to set up the runtime for oapi-codegen
func initAPIRuntime() {
	// Opt-in to OpenAPI 3.1 type arrays
	runtime.NarrowUnionNumericFormats = true
}

func SetupAndRun(config *core.Config, version string, async bool) *IngestorWebServerImplemenation {
	initAPIRuntime()

	if !strings.HasSuffix(config.Scicat.Host, "v3") {
		panic(fmt.Sprintf("Only Scicat API v3 is supported. No v3 suffix found in API path. Got '%s'", config.Scicat.Host))
	}

	for location := range config.WebServer.CollectionLocations {
		if strings.Contains(location, "/") {
			panic(fmt.Sprintf("Invalid name `%s` in 'Collectionlocations`. Cannot be a path or contain `/`", location))
		}
	}

	ctx := context.Background()

	u, foundName := os.LookupEnv("INGESTOR_SERVICE_USER_NAME")
	p, foundPass := os.LookupEnv("INGESTOR_SERVICE_USER_PASS")
	var serviceAcc *core.UserCreds = nil

	if foundName && foundPass {
		serviceAcc = &core.UserCreds{
			Username: u,
			Password: p,
		}
	}

	totalConcurrencyLimit := config.WebServer.GlobalConcurrencyLimit
	mainPool := pond.NewPool(totalConcurrencyLimit)

	extractorHandler := metadataextractor.NewExtractorHandler(config.MetadataExtractors)

	metadataExtractorPool := metadatatasks.NewTaskPoolFromPool(config.WebServer.MetadataExtJobsConf.ConcurrencyLimit,
		config.WebServer.MetadataExtJobsConf.QueueSize,
		extractorHandler,
		&mainPool)

	taskQueuePool := mainPool.NewSubpool(config.Transfer.ConcurrencyLimit, pond.WithNonBlocking(true))
	taskQueue := core.NewTaskQueueFromPool(ctx, *config, core.NewLoggingNotifier(), serviceAcc, taskQueuePool)

	if strings.ToLower(config.Transfer.Method) == "s3" {
		s3PoolSize := min(config.Transfer.S3.PoolSize, totalConcurrencyLimit-config.WebServer.MetadataExtJobsConf.ConcurrencyLimit-config.WebServer.ConcurrencyLimit)
		s3upload.InitHTTPUploaderWithPool(mainPool.NewSubpool(s3PoolSize))
	}

	ingestor, err := NewIngestorWebServer(version, taskQueue, extractorHandler, metadataExtractorPool, config.WebServer)
	if err != nil {
		log.Fatal(err)
	}

	slog.Info("Ingestor started and listening", "port", config.WebServer.Port, "version", version)
	s := NewIngestorServer(ingestor, config.WebServer.Port)

	if async {
		go func() {
			log.Fatal(s.ListenAndServe())
		}()
	} else {
		log.Fatal(s.ListenAndServe())

	}

	return ingestor
}
