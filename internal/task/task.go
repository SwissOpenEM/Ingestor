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

type TaskStatus struct {
	BytesTransferred int
	BytesTotal       int
	FilesTransferred int
	FilesTotal       int
	Failed           bool
	Started          bool
	Finished         bool
	StatusMessage    string
}

type TransferTask struct {
	// DatasetFolderId   uuid.UUID
	DatasetFolder
	datasetId       string
	fileList        []datasetIngestor.Datafile
	DatasetMetadata map[string]interface{}
	TransferMethod  TransferMethod
	Cancel          context.CancelFunc
	status          *TaskStatus
	statusLock      *sync.RWMutex
	transferObjects map[string]interface{}
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
		status:          &TaskStatus{},
		statusLock:      &sync.RWMutex{},
	}
}

func (t *TransferTask) GetStatus() TaskStatus {
	t.statusLock.RLock()
	defer t.statusLock.RUnlock()
	copy := *t.status
	return copy
}

func (t *TransferTask) UpdateStatus(options ...StatusOption) {
	t.statusLock.Lock()
	defer t.statusLock.Unlock()
	for _, option := range options {
		option(t)
	}
}

func SetBytesTransferred(b int) StatusOption {
	return func(t *TransferTask) {
		t.status.BytesTransferred = b
	}
}

func SetBytesTotal(b int) StatusOption {
	return func(t *TransferTask) {
		t.status.BytesTotal = b
	}
}

func SetFilesTransferred(f int) StatusOption {
	return func(t *TransferTask) {
		t.status.FilesTransferred = f
	}
}

func SetFilesTotal(f int) StatusOption {
	return func(t *TransferTask) {
		t.status.FilesTotal = f
	}
}

func SetFailed(f bool) StatusOption {
	return func(t *TransferTask) {
		t.status.Failed = f
	}
}

func SetStarted(s bool) StatusOption {
	return func(t *TransferTask) {
		t.status.Started = s
	}
}

func SetFinished(f bool) StatusOption {
	return func(t *TransferTask) {
		t.status.Finished = f
	}
}

func SetStatusMessage(m string) StatusOption {
	return func(t *TransferTask) {
		t.status.StatusMessage = m
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
