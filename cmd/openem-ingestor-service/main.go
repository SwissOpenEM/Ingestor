package main

import (
	"context"
	"fmt"
	"log"
	"log/slog"
	"os"

	core "github.com/SwissOpenEM/Ingestor/internal/core"
	"github.com/SwissOpenEM/Ingestor/internal/metadataextractor"
	"github.com/SwissOpenEM/Ingestor/internal/webserver"
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

	ctx := context.Background()

	var serviceUser *core.UserCreds = nil
	u, foundName := os.LookupEnv("INGESTOR_SERVICE_USER_NAME")
	p, foundPass := os.LookupEnv("INGESTOR_SERVICE_USER_PASS")
	if foundName && foundPass {
		serviceUser = &core.UserCreds{
			Username: u,
			Password: p,
		}
	} else {
		panic(fmt.Errorf("service user was not set in env vars"))
	}

	tq := core.TaskQueue{
		Config:      config,
		AppContext:  ctx,
		Notifier:    core.NewLoggingNotifier(),
		ServiceUser: serviceUser,
	}
	tq.Startup()

	eh := metadataextractor.NewExtractorHandler(config.MetadataExtractors)

	ingestor, err := webserver.NewIngestorWebServer(version, &tq, eh, config.WebServer)
	if err != nil {
		log.Fatal(err)
	}
	s := webserver.NewIngesterServer(ingestor, config.WebServer.Port)
	log.Fatal(s.ListenAndServe())
}
