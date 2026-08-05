package main

import (
	_ "embed"
	"fmt"
	"os"
	"path"
	"runtime"
	"time"

	"log/slog"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/widget"
	"github.com/SwissOpenEM/Ingestor/internal/core"
	"github.com/SwissOpenEM/Ingestor/internal/transfertask"
	"github.com/SwissOpenEM/Ingestor/internal/ui"
	"github.com/SwissOpenEM/Ingestor/internal/webserver"
	"gopkg.in/natefinch/lumberjack.v2"
	"gopkg.in/yaml.v2"
)

//go:embed openem.ico
var iconData []byte

func convertLogLevel(logLevel string) slog.Level {
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
	return level
}

func setupLogging(logLevel string, widget *ui.LogWidget) {

	level := convertLogLevel(logLevel)
	opts := &slog.HandlerOptions{Level: level}
	cacheDir, _ := os.UserCacheDir()
	logDir := path.Join(cacheDir, "openem-ingestor")
	os.MkdirAll(logDir, 0666)
	logPath := path.Join(logDir, "log.txt")

	handler := ui.NewWidgetHandler(widget, opts.Level.Level())
	rotator := &lumberjack.Logger{
		Filename:   logPath,
		MaxSize:    10,
		MaxBackups: 5,
		MaxAge:     28,
		Compress:   true,
	}
	jsonHandler := slog.NewTextHandler(rotator, &slog.HandlerOptions{})
	multiHandler := slog.NewMultiHandler(handler, jsonHandler)
	slog.SetDefault(slog.New(multiHandler))
}

func main() {
	logWidget := ui.NewLogWidget()
	setupLogging("Info", logWidget)

	var config core.Config
	configFileReader := core.NewConfigReader()
	var err error
	if config, err = configFileReader.ReadConfig(core.DefaultConfigFileName()); err != nil {
		slog.Error("Reading config failed", "file", configFileReader.GetCurrentConfigFilePath(), "error", err)
		panic(fmt.Errorf("failed to read config file: %w", err))
	}

	slog.Info("Config read", "Filepath", configFileReader.GetCurrentConfigFilePath())
	slog.Info("Connected", "Scicat", config.Scicat.Host, "S3 Endpoint", config.Transfer.S3.Endpoint)

	slog.SetLogLoggerLevel(convertLogLevel(config.WebServer.LogLevel))
	slog.Info("Loglevel changed", "Loglevel", config.WebServer.LogLevel)

	configData, _ := yaml.Marshal(configFileReader.GetFullConfig())
	println(string(configData))

	a := app.New()
	m := a.Metadata()
	var version = m.Version
	if version == "" {
		version = "DEVELOPMENT_VERSION"
	}

	ingestorImplementation := webserver.SetupAndRun(&config, version, true)

	a.Settings().SetTheme(ui.PsiTheme{})
	w := a.NewWindow(fmt.Sprintf("OpenEM Ingestor %s", version))

	var tasks []*transfertask.TransferTask

	ui := ui.NewTaskListUI(tasks)

	go func() {
		for range time.Tick(1000 * time.Millisecond) {
			go func() {
				fyne.DoAndWait(func() {
					tasks, _ := ingestorImplementation.GetTasks()
					ui.SetTasks(tasks)
					ui.Refresh()
				})
			}()
		}
		// not sure it is really needed
		runtime.KeepAlive(ingestorImplementation)
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
	// header
	// bottom
	// left
	// right
	// center
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

	// Keep app running in system tray
	w.SetCloseIntercept(func() { w.Hide() })

	w.ShowAndRun()
	slog.Info("Shutting down ingestor")

}
