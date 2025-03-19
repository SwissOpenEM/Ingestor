package transfertask

import (
	"context"
	"sync"

	"github.com/paulscherrerinstitute/scicat-cli/v3/datasetIngestor"
)

type TransferMethod int

const (
	TransferS3 TransferMethod = iota + 1
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

type TaskDetails struct {
	BytesTransferred int
	BytesTotal       int
	FilesTransferred int
	FilesTotal       int
	Status           Status
	Message          string
}

type Status int

const (
	Waiting Status = iota
	Transferring
	Finished
	Failed
	Cancelled
)

func (i *Status) ToStr() string {
	switch *i {
	case Waiting:
		return "waiting"
	case Transferring:
		return "transferring"
	case Finished:
		return "finished"
	case Failed:
		return "failed"
	default:
		return "invalid status"
	}
}

type TransferTask struct {
	DatasetFolder   DatasetFolder
	datasetId       string
	fileList        []datasetIngestor.Datafile
	TransferMethod  TransferMethod
	Context         context.Context
	Cancel          context.CancelFunc
	details         *TaskDetails
	statusLock      *sync.RWMutex
	transferObjects map[string]interface{}
}

type Result struct {
	Elapsed_seconds int
	Dataset_PID     string
	Error           error
}

func CreateTransferTask(datasetId string, fileList []datasetIngestor.Datafile, datasetFolder DatasetFolder, transferMethod TransferMethod, transferObjects map[string]interface{}, cancel context.CancelFunc) TransferTask {
	totalBytes := 0
	for _, file := range fileList {
		totalBytes += int(file.Size)
	}
	return TransferTask{
		datasetId:       datasetId,
		fileList:        fileList,
		DatasetFolder:   datasetFolder,
		TransferMethod:  transferMethod,
		transferObjects: transferObjects,
		Cancel:          cancel,
		details: &TaskDetails{
			BytesTransferred: 0,
			BytesTotal:       totalBytes,
			FilesTransferred: 0,
			FilesTotal:       len(fileList),
			Status:           Waiting,
			Message:          "in waiting list",
		},
		statusLock: &sync.RWMutex{},
	}
}

func (t *TransferTask) GetDetails() TaskDetails {
	t.statusLock.RLock()
	defer t.statusLock.RUnlock()
	copy := *t.details
	return copy
}

func (t *TransferTask) Queued() {
	t.statusLock.Lock()
	defer t.statusLock.Unlock()
	if t.details.Status != Waiting {
		return
	}
	t.details.Message = "queued"
}

func (t *TransferTask) TransferStarted() {
	t.statusLock.Lock()
	defer t.statusLock.Unlock()
	if t.details.Status != Waiting {
		return
	}
	t.details.Status = Transferring
	t.details.Message = "transferring files"
}

func (t *TransferTask) UpdateProgress(bytesTransferred *int, filesTransferred *int) {
	t.statusLock.Lock()
	defer t.statusLock.Unlock()
	if t.details.Status != Transferring {
		return
	}
	if bytesTransferred != nil {
		t.details.BytesTransferred = *bytesTransferred
	}
	if filesTransferred != nil {
		t.details.FilesTransferred = *filesTransferred
	}
}

func (t *TransferTask) Finished() {
	t.statusLock.Lock()
	defer t.statusLock.Unlock()
	if t.details.Status != Transferring {
		return
	}
	t.details.Status = Finished
	t.details.Message = "transfer has finished"
}

func (t *TransferTask) Failed(msg string) {
	t.statusLock.Lock()
	defer t.statusLock.Unlock()
	if t.details.Status == Finished ||
		t.details.Status == Cancelled ||
		t.details.Status == Failed {
		return
	}
	t.details.Status = Failed
	t.details.Message = msg
}

func (t *TransferTask) Cancelled(msg string) {
	t.statusLock.Lock()
	defer t.statusLock.Unlock()
	if t.details.Status == Failed ||
		t.details.Status == Cancelled ||
		t.details.Status == Finished {
		return
	}
	t.details.Status = Cancelled
	t.details.Message = msg
}

func (t *TransferTask) GetDatasetId() string {
	return t.datasetId
}

func (t *TransferTask) GetFileList() []datasetIngestor.Datafile {
	return t.fileList
}

func (t *TransferTask) GetTransferObject(name string) interface{} {
	return t.transferObjects[name]
}
