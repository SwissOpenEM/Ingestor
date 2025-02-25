package task

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
	DatasetMetadata map[string]interface{}
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

type StatusOption func(t *TransferTask)

func CreateTransferTask(datasetId string, fileList []datasetIngestor.Datafile, datasetFolder DatasetFolder, metadata map[string]interface{}, transferMethod TransferMethod, transferObjects map[string]interface{}, cancel context.CancelFunc) TransferTask {
	return TransferTask{
		datasetId:       datasetId,
		fileList:        fileList,
		DatasetFolder:   datasetFolder,
		DatasetMetadata: metadata,
		TransferMethod:  transferMethod,
		transferObjects: transferObjects,
		Cancel:          cancel,
		details:         &TaskDetails{},
		statusLock:      &sync.RWMutex{},
	}
}

func (t *TransferTask) GetDetails() TaskDetails {
	t.statusLock.RLock()
	defer t.statusLock.RUnlock()
	copy := *t.details
	return copy
}

func (t *TransferTask) UpdateDetails(options ...StatusOption) {
	t.statusLock.Lock()
	defer t.statusLock.Unlock()
	for _, option := range options {
		option(t)
	}
}

func SetBytesTransferred(b int) StatusOption {
	return func(t *TransferTask) {
		t.details.BytesTransferred = b
	}
}

func SetBytesTotal(b int) StatusOption {
	return func(t *TransferTask) {
		t.details.BytesTotal = b
	}
}

func SetFilesTransferred(f int) StatusOption {
	return func(t *TransferTask) {
		t.details.FilesTransferred = f
	}
}

func SetFilesTotal(f int) StatusOption {
	return func(t *TransferTask) {
		t.details.FilesTotal = f
	}
}

func SetStatus(s Status) StatusOption {
	return func(t *TransferTask) {
		t.details.Status = s
	}
}

func SetMessage(m string) StatusOption {
	return func(t *TransferTask) {
		t.details.Message = m
	}
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
