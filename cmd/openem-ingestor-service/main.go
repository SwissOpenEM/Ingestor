package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"

	"log/slog"

	core "github.com/SwissOpenEM/Ingestor/internal/core"
	"github.com/SwissOpenEM/Ingestor/internal/metadataextractor"
	"github.com/SwissOpenEM/Ingestor/internal/s3upload"
	"github.com/SwissOpenEM/Ingestor/internal/webserver"
	"github.com/SwissOpenEM/Ingestor/internal/webserver/metadatatasks"
	"github.com/alitto/pond/v2"
	"gopkg.in/yaml.v2"
)

// String can be overwritten by using linker flags: -ldflags "-X main.version=VERSION"
var version string = "DEVELOPMENT_VERSION"

func setupLogging(logLevel string) {
	level := slog.LevelDebug
	switch logLevel {
	case "Info":
		level = slog.LevelInfo
	case "Debug":
		level = slog.LevelDebug
	case "Error":
		level = slog.LevelError
	case "Warning":
		level = slog.LevelWarn
	}

	opts := &slog.HandlerOptions{Level: level}
	h := slog.NewTextHandler(os.Stdout, opts)
	slog.SetDefault(slog.New(h))
}

func main() {
	slog.Info("Starting ingestor service", "Version", version)

	var config core.Config
	configFileReader := core.NewConfigReader()
	var err error
	if config, err = configFileReader.ReadConfig(core.DefaultConfigFileName()); err != nil {
		slog.Info("Reading config", "file", configFileReader.GetCurrentConfigFilePath())
		panic(fmt.Errorf("failed to read config file: %w", err))
	}

	slog.Info("Config read", "Filepath", configFileReader.GetCurrentConfigFilePath())

	configData, _ := yaml.Marshal(configFileReader.GetFullConfig())
	println(string(configData))

	setupLogging(config.WebServer.LogLevel)

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
	taskqueue := core.NewTaskQueueFromPool(ctx, config, core.NewLoggingNotifier(), serviceAcc, taskQueuePool)

	if strings.ToLower(config.Transfer.Method) == "s3" {
		s3PoolSize := min(config.Transfer.S3.PoolSize, totalConcurrencyLimit-config.WebServer.MetadataExtJobsConf.ConcurrencyLimit-config.WebServer.ConcurrencyLimit)
		s3upload.InitHTTPUploaderWithPool(mainPool.NewSubpool(s3PoolSize))
	}

	ingestor, err := webserver.NewIngestorWebServer(version, taskqueue, extractorHandler, metadataExtractorPool, config.WebServer)
	if err != nil {
		log.Fatal(err)
	}

	slog.Info("Ingestor started and listening", "port", config.WebServer.Port, "version", version)
	s := webserver.NewIngesterServer(ingestor, config.WebServer.Port)
	log.Fatal(s.ListenAndServe())
}
