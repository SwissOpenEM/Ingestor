package main

import (
	"context"
	"log"

	core "github.com/SwissOpenEM/Ingestor/internal/core"
	webserver "github.com/SwissOpenEM/Ingestor/internal/webserver"

	"github.com/google/uuid"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

type WailsNotifier struct {
	AppContext context.Context
}

func (w *WailsNotifier) OnTaskScheduled(id uuid.UUID) {
	runtime.EventsEmit(w.AppContext, "upload-scheduled", id)
}
func (w *WailsNotifier) OnTaskCanceled(id uuid.UUID) {
	runtime.EventsEmit(w.AppContext, "upload-canceled", id)
}
func (w *WailsNotifier) OnTaskRemoved(id uuid.UUID) {
	runtime.EventsEmit(w.AppContext, "folder-removed", id)
}
func (w *WailsNotifier) OnTaskFailed(id uuid.UUID, err error) {
	runtime.EventsEmit(w.AppContext, "upload-failed", id, err.Error())
}
func (w *WailsNotifier) OnTaskCompleted(id uuid.UUID, seconds_elapsed int) {
	runtime.EventsEmit(w.AppContext, "upload-completed", id, seconds_elapsed)
}
func (w *WailsNotifier) OnTaskProgress(id uuid.UUID, current_file int, total_files int, elapsed_seconds int) {
	runtime.EventsEmit(w.AppContext, "progress-update", id, current_file, total_files, elapsed_seconds)
}

// App struct
type App struct {
	ctx       context.Context
	taskqueue core.TaskQueue
	config    core.Config
	version   string
}

// NewApp creates a new App application struct
func NewApp(config core.Config, version string) *App {
	return &App{config: config, version: version}
}

// Show prompt before closing the app
func (b *App) beforeClose(ctx context.Context) (prevent bool) {
	dialog, err := runtime.MessageDialog(ctx, runtime.MessageDialogOptions{
		Type:    runtime.QuestionDialog,
		Title:   "Quit?",
		Message: "Are you sure you want to quit? This will stop all pending downloads.",
	})

	if err != nil {
		return false
	}
	return dialog != "Yes"
}

// startup is called when the app starts. The context is saved
// so we can call the runtime methods
func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
	a.taskqueue = core.TaskQueue{Config: a.config,
		AppContext: a.ctx,
		Notifier:   &WailsNotifier{AppContext: a.ctx},
	}
	a.taskqueue.Startup()

	go func(port int) {
		ingestor := webserver.NewIngestorWebServer(a.version)
		s := webserver.NewIngesterServer(ingestor, port)
		log.Fatal(s.ListenAndServe())
	}(a.config.Misc.Port)
}

func (a *App) SelectFolder() {
	folder, err := core.SelectFolder(a.ctx)
	if err != nil {
		return
	}

	err = a.taskqueue.CreateTask(folder)
	if err != nil {
		return
	}
}

func (a *App) CancelTask(id uuid.UUID) {
	a.taskqueue.CancelTask(id)
}
func (a *App) RemoveTask(id uuid.UUID) {
	a.taskqueue.RemoveTask(id)
}

func (a *App) ScheduleTask(id uuid.UUID) {

	a.taskqueue.ScheduleTask(id)
}
