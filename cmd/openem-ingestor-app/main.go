package main

import (
	"context"
	_ "embed"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"log/slog"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/widget"
	core "github.com/SwissOpenEM/Ingestor/internal/core"
	"github.com/SwissOpenEM/Ingestor/internal/metadataextractor"
	"github.com/SwissOpenEM/Ingestor/internal/s3upload"
	"github.com/SwissOpenEM/Ingestor/internal/transfertask"
	"github.com/SwissOpenEM/Ingestor/internal/ui"
	"github.com/SwissOpenEM/Ingestor/internal/webserver"
	"github.com/SwissOpenEM/Ingestor/internal/webserver/metadatatasks"
	"github.com/alitto/pond/v2"
	"gopkg.in/yaml.v2"
)

// String can be overwritten by using linker flags: -ldflags "-X main.version=VERSION"
var version string = "DEVELOPMENT_VERSION"

var gTaskQueue *core.TaskQueue

func setupLogging(logLevel string, widget *ui.LogWidget) {
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
	// h := slog.NewTextHandler(os.Stdout, opts)

	handler := ui.NewWidgetHandler(widget, opts.Level.Level())
	slog.SetDefault(slog.New(handler))
}

func main() {
	slog.Info("Starting ingestor service", "Version", version)
	logWidget := ui.NewLogWidget()
	setupLogging("Info", logWidget)

	var config core.Config
	configFileReader := core.NewConfigReader()
	var err error
	if config, err = configFileReader.ReadConfig(core.DefaultConfigFileName()); err != nil {
		slog.Info("Reading config", "file", configFileReader.GetCurrentConfigFilePath())
		panic(fmt.Errorf("failed to read config file: %w", err))
	}

	slog.Info("Config read", "Filepath", configFileReader.GetCurrentConfigFilePath())

	setupLogging(config.WebServer.LogLevel, logWidget)

	configData, _ := yaml.Marshal(configFileReader.GetFullConfig())
	println(string(configData))

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
	taskQueue := core.NewTaskQueueFromPool(ctx, config, core.NewLoggingNotifier(), serviceAcc, taskQueuePool)
	gTaskQueue = taskQueue

	if strings.ToLower(config.Transfer.Method) == "s3" {
		s3PoolSize := min(config.Transfer.S3.PoolSize, totalConcurrencyLimit-config.WebServer.MetadataExtJobsConf.ConcurrencyLimit-config.WebServer.ConcurrencyLimit)
		s3upload.InitHTTPUploaderWithPool(mainPool.NewSubpool(s3PoolSize))
	}

	ingestor, err := webserver.NewIngestorWebServer(version, taskQueue, extractorHandler, metadataExtractorPool, config.WebServer)
	if err != nil {
		log.Fatal(err)
	}

	slog.Info("Ingestor started and listening", "port", config.WebServer.Port, "version", version)
	s := webserver.NewIngesterServer(ingestor, config.WebServer.Port)

	go func() {
		log.Fatal(s.ListenAndServe())
	}()

	a := app.New()

	a.Settings().SetTheme(ui.PsiTheme{})
	w := a.NewWindow(fmt.Sprintf("OpenEM Ingestor %s", version))

	// logger := slog.New(handler)

	var tasks []*transfertask.TransferTask

	ui := ui.NewTaskListUI(tasks)

	go func() {
		for range time.Tick(1000 * time.Millisecond) {
			go func() {
				fyne.DoAndWait(func() {
					tasks, _ := gTaskQueue.GetTasks()
					ui.SetTasks(tasks)
					ui.Refresh()
				})
			}()

		}
	}()

	header := widget.NewLabelWithStyle(
		"Transfer Tasks",
		fyne.TextAlignLeading,
		fyne.TextStyle{Bold: true},
	)

	logHeader := widget.NewLabelWithStyle(
		"Output",
		fyne.TextAlignLeading,
		fyne.TextStyle{Bold: true},
	)

	defaultSize := fyne.NewSize(800, 600)
	w.Resize(defaultSize)
	menu := fyne.NewMenu(fmt.Sprintf("OpenEM Ingestor %s", version),
		fyne.NewMenuItem("Show", func() {
			w.Resize(defaultSize)
			w.Show()
			w.RequestFocus()
		}),
		fyne.NewMenuItem("Hide", func() {
			w.Hide()
		}),
	)

	res := fyne.NewStaticResource("openem", iconData)
	a.SetIcon(res)
	if desk, ok := a.(desktop.App); ok {
		desk.SetSystemTrayIcon(res)
		desk.SetSystemTrayMenu(menu)
	}
	// // header
	// // bottom
	// // left
	// // right
	// // center
	w.SetContent(container.NewBorder(
		nil,
		container.NewBorder(
			container.NewVBox(
				logHeader,
				widget.NewSeparator(),
			),
			logWidget, nil, nil, nil,
		),
		nil,
		nil,
		container.NewBorder(
			container.NewVBox(
				header,
				widget.NewSeparator(),
			),
			nil, nil, nil, ui.Container(),
		),
	))

	w.SetCloseIntercept(func() { w.Hide() })

	w.ShowAndRun()

}

//go:embed openem.ico
var iconData []byte
