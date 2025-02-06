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

	"github.com/spf13/viper"
)

// String can be overwritten by using linker flags: -ldflags "-X main.version=VERSION"
var version string = "DEVELOPMENT_VERSION"

func main() {
	slog.Info("", "Version", version)

	var config core.Config
	var err error
	if config, err = core.ReadConfig(core.DefaultConfigFileName()); err != nil {
		slog.Info("Config file read", "file", viper.ConfigFileUsed())
		panic(fmt.Errorf("failed to read config file: %w", err))
	}

	log.Println(core.GetFullConfig())
	log.Printf("Config file used: %s", core.GetCurrentConfigFilePath())

	ctx := context.Background()

	// setup globus if we have a refresh token
	if config.Transfer.Globus.RefreshToken != "" {
		core.GlobusLoginWithRefreshToken(config.Transfer.Globus)
	}

	var serviceUser *core.UserCreds = nil
	u, foundName := os.LookupEnv("INGESTOR_SERVICE_USER_NAME")
	p, foundPass := os.LookupEnv("INGESTOR_SERVICE_USER_PASS")
	if foundName && foundPass {
		serviceUser = &core.UserCreds{
			Username: u,
			Password: p,
		}
	}

	tq := core.TaskQueue{
		Config:      config,
		AppContext:  ctx,
		Notifier:    &core.LoggingNotifier{},
		ServiceUser: serviceUser,
	}
	tq.Startup()

	eh := metadataextractor.NewExtractorHandler(config.MetadataExtractors)

	ingestor, err := webserver.NewIngestorWebServer(version, &tq, eh, config.WebServer)
	if err != nil {
		log.Fatal(err)
	}
	s := webserver.NewIngesterServer(ingestor, config.Misc.Port)
	log.Fatal(s.ListenAndServe())
}
