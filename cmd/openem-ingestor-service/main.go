package main

import (
	"context"
	"fmt"
	"log"

	core "github.com/SwissOpenEM/Ingestor/internal/core"
	"github.com/SwissOpenEM/Ingestor/internal/webserver"
	"github.com/google/uuid"

	"github.com/spf13/viper"
)

// String can be overwritten by using linker flags: -ldflags "-X main.version=VERSION"
var version string = "DEVELOPMENT_VERSION"

type DummyNotifier struct{}

func (n *DummyNotifier) OnTaskScheduled(id uuid.UUID)                     {}
func (n *DummyNotifier) OnTaskCanceled(id uuid.UUID)                      {}
func (n *DummyNotifier) OnTaskAdded(id uuid.UUID, folder string)          {}
func (n *DummyNotifier) OnTaskRemoved(id uuid.UUID)                       {}
func (n *DummyNotifier) OnTaskFailed(id uuid.UUID, err error)             {}
func (n *DummyNotifier) OnTaskCompleted(id uuid.UUID, elapsedSeconds int) {}
func (n *DummyNotifier) OnTaskProgress(id uuid.UUID, currentFile int, totalFiles int, elapsedSeconds int) {
}

func main() {
	log.Printf("Version %s", version)

	if err := core.ReadConfig(); err != nil {
		panic(fmt.Errorf("failed to read config file: %w", err))
	}
	log.Printf("Config file used: %s", viper.ConfigFileUsed())
	log.Println(viper.AllSettings())

	ctx := context.Background()
	config, err := core.GetConfig()
	if err != nil {
		log.Fatalf("could not retrieve config: %s\n", err.Error())
	}

	taskqueue := core.TaskQueue{
		Config:     config,
		AppContext: ctx,
		Notifier:   &DummyNotifier{},
	}
	taskqueue.Startup()

	ingestor := webserver.NewIngestorWebServer(version, &taskqueue)
	s := webserver.NewIngesterServer(ingestor, config.Misc.Port)
	log.Fatal(s.ListenAndServe())
}
