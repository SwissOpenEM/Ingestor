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

func (t *TransferTask) SetDetails(
	bytesTransferred *int,
	bytesTotal *int,
	filesTransferred *int,
	filesTotal *int,
	status *Status,
	message *string,
) {
	t.statusLock.Lock()
	defer t.statusLock.Unlock()
	if bytesTransferred != nil {
		t.details.BytesTransferred = *bytesTransferred
	}
	if bytesTotal != nil {
		t.details.BytesTotal = *bytesTotal
	}
	if filesTransferred != nil {
		t.details.FilesTransferred = *filesTransferred
	}
	if filesTotal != nil {
		t.details.FilesTotal = *filesTotal
	}
	if status != nil {
		t.details.Status = *status
	}
	if message != nil {
		t.details.Message = *message
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
