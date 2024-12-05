package main

import (
	"context"
	"fmt"
	"log"
	"log/slog"

	core "github.com/SwissOpenEM/Ingestor/internal/core"
	"github.com/SwissOpenEM/Ingestor/internal/metadataextractor"
	"github.com/SwissOpenEM/Ingestor/internal/webserver"

	"github.com/spf13/viper"
)

// String can be overwritten by using linker flags: -ldflags "-X main.version=VERSION"
var version string = "DEVELOPMENT_VERSION"

func main() {
	slog.Info("", "Version", version)

	var config core.Config
	var err error
	if config, err = core.ReadConfig(core.DefaultConfigFileName()); err != nil {
		panic(fmt.Errorf("failed to read config file: %w", err))
	}

	slog.Info("Config file read", "file", viper.ConfigFileUsed())
	log.Println(viper.AllSettings())

	ctx := context.Background()

	taskqueue := core.TaskQueue{
		Config:     config,
		AppContext: ctx,
		Notifier:   &core.LoggingNotifier{},
	}
	taskqueue.Startup()

	extractorHander := metadataextractor.NewExtractorHandler(config.MetadataExtractors)

	ingestor, err := webserver.NewIngestorWebServer(version, &taskqueue, extractorHander, config.WebServerAuth, config.WebServerPaths)
	if err != nil {
		log.Fatal(err)
	}
	s := webserver.NewIngesterServer(ingestor, config.Misc.Port)
	log.Fatal(s.ListenAndServe())
}
