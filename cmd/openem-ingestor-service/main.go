package main

import (
	"context"
	"fmt"
	"log"
	"log/slog"

	core "github.com/SwissOpenEM/Ingestor/internal/core"
	"github.com/SwissOpenEM/Ingestor/internal/webserver"

	"github.com/spf13/viper"
)

// String can be overwritten by using linker flags: -ldflags "-X main.version=VERSION"
var version string = "DEVELOPMENT_VERSION"

func main() {
	slog.Info("", "Version", version)

	if err := core.ReadConfig(); err != nil {
		panic(fmt.Errorf("failed to read config file: %w", err))
	}
	slog.Info("Config file read", "file", viper.ConfigFileUsed())
	log.Println(viper.AllSettings())

	ctx := context.Background()
	config, err := core.GetConfig()
	if err != nil {
		log.Fatalf("could not retrieve config: %s\n", err.Error())
	}

	taskqueue := core.TaskQueue{
		Config:     config,
		AppContext: ctx,
		Notifier:   &core.LoggingNotifier{},
	}
	taskqueue.Startup()

	ingestor := webserver.NewIngestorWebServer(version, &taskqueue)
	s := webserver.NewIngesterServer(ingestor, config.Misc.Port)
	log.Fatal(s.ListenAndServe())
}
