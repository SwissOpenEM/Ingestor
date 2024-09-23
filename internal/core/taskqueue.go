package core

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
)

type TaskQueue struct {
	datasetSourceFolders sync.Map           // Datastructure to store all the upload requests
	inputChannel         chan IngestionTask // Requests to upload data are put into this channel
	resultChannel        chan TaskResult    // The result of the upload is put into this channel
	AppContext           context.Context
	Config               Config
	Notifier             ProgressNotifier
}

type TransferMethods int

const (
	TransferS3 TransferMethods = iota + 1
	TransferGlobus
)

type TransferOptions struct {
	S3_endpoint string
	S3_Bucket   string
	Md5checksum bool
}

type TaskTransferConfig struct {
	S3TransferConfig
	GlobusTransferConfig
}

type IngestionTask struct {
	// DatasetFolderId   uuid.UUID
	DatasetFolder
	TransferMethod TransferMethods
	Cancel         context.CancelFunc
}

type TaskResult struct {
	Elapsed_seconds int
	Dataset_PID     string
	Error           error
}

func (w *TaskQueue) Startup() {

	w.inputChannel = make(chan IngestionTask)
	w.resultChannel = make(chan TaskResult)

	// start multiple go routines/workers that will listen on the input channel
	for worker := 1; worker <= w.Config.Misc.ConcurrencyLimit; worker++ {
		go w.startWorker()
	}

}

func (w *TaskQueue) CreateTask(folder DatasetFolder) error {
	var transferMethod TransferMethods
	switch strings.ToLower(w.Config.Transfer.Method) {
	case "globus":
		transferMethod = TransferGlobus
	case "s3":
		transferMethod = TransferS3
	}

	task := IngestionTask{
		DatasetFolder:  folder,
		TransferMethod: transferMethod,
	}
	_, found := w.datasetSourceFolders.Load(task.DatasetFolder.Id)
	if found {
		return errors.New("key exists")
	}
	w.datasetSourceFolders.Store(task.DatasetFolder.Id, task)

	w.Notifier.OnTaskAdded(task.DatasetFolder.Id, task.DatasetFolder.FolderPath)
	return nil
}

// Go routine that listens on the channel continously for upload requests and executes uploads.
func (w *TaskQueue) startWorker() {
	for ingestionTask := range w.inputChannel {
		task_context, cancel := context.WithCancel(w.AppContext)

		ingestionTask.Cancel = cancel

		result := w.IngestDataset(task_context, ingestionTask)
		w.resultChannel <- result
	}
}

func (w *TaskQueue) CancelTask(id uuid.UUID) {
	value, found := w.datasetSourceFolders.Load(id)
	if found {
		f := value.(IngestionTask)
		if f.Cancel != nil {
			f.Cancel()
		}
		w.Notifier.OnTaskCanceled(id)
	}
}

func (w *TaskQueue) RemoveTask(id uuid.UUID) {
	value, found := w.datasetSourceFolders.Load(id)
	if found {
		f := value.(IngestionTask)
		if f.Cancel != nil {
			f.Cancel()
		}
		w.datasetSourceFolders.Delete(id)
		w.Notifier.OnTaskRemoved(id)
	}
}

func (w *TaskQueue) ScheduleTask(id uuid.UUID) {

	value, found := w.datasetSourceFolders.Load(id)
	if !found {
		fmt.Println("Scheduling upload failed for: ", id)
		return
	}

	task := value.(IngestionTask)

	// Go routine to handle result and errors
	go func(id uuid.UUID) {
		taskResult := <-w.resultChannel
		if taskResult.Error != nil {
			w.Notifier.OnTaskFailed(id, taskResult.Error)
			println(fmt.Sprint(taskResult.Error))
		} else {
			w.Notifier.OnTaskCompleted(id, taskResult.Elapsed_seconds)
			println(taskResult.Dataset_PID, taskResult.Elapsed_seconds)
		}
	}(task.DatasetFolder.Id)

	// Go routine to schedule the upload asynchronously
	go func(folder DatasetFolder) {
		fmt.Println("Scheduled upload for: ", folder)
		w.Notifier.OnTaskScheduled(folder.Id)

		// this channel is read by the go routines that does the actual upload
		w.inputChannel <- task
	}(task.DatasetFolder)

}

func TestIngestionFunction(task_context context.Context, task IngestionTask, config Config, notifier ProgressNotifier) (string, error) {

	start := time.Now()

	for i := 0; i < 10; i++ {
		time.Sleep(time.Second * 1)
		now := time.Now()
		elapsed := now.Sub(start)
		notifier.OnTaskProgress(task.DatasetFolder.Id, i+1, 10, int(elapsed.Seconds()))
	}
	return "1", nil

}

func (w *TaskQueue) IngestDataset(task_context context.Context, task IngestionTask) TaskResult {
	start := time.Now()
	// datasetPID, _ := TestIngestionFunction(task_context, task, w.Config, w.Notifier)
	datasetPID, err := IngestDataset(task_context, task, w.Config, w.Notifier)
	end := time.Now()
	elapsed := end.Sub(start)
	return TaskResult{Dataset_PID: datasetPID, Elapsed_seconds: int(elapsed.Seconds()), Error: err}
}
