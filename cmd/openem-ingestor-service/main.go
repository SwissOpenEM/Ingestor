package main

import (
	"context"
	"crypto/aes"
	"crypto/rand"
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

	// generate key for AES
	key := make([]byte, 32)
	_, err = rand.Read(key)
	if err != nil {
		panic("can't generate AES key")
	}

	// create AES encryption for state cookie encryption
	aes, err := aes.NewCipher(key)
	if err != nil {
		panic("can't create aes cipher")
	}

	ingestor := webserver.NewIngestorWebServer(version, &taskqueue, &config.Oauth, aes)
	s := webserver.NewIngesterServer(ingestor, config.Misc.Port)
	log.Fatal(s.ListenAndServe())
}
