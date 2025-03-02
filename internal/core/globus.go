package core

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/SwissOpenEM/Ingestor/internal/task"
	"github.com/SwissOpenEM/globus"
	"github.com/google/uuid"
	"github.com/paulscherrerinstitute/scicat-cli/v3/datasetIngestor"
)

/*func GlobusHealthCheck() error {
	// NOTE: this is not a proper health check and takes a long time to finish (~900ms)
	_, err := globusClient.TransferGetTaskList(0, 1)
	return err
}*/

func globusCheckTransfer(client *globus.GlobusClient, globusTaskId string) (bytesTransferred int, filesTransferred int, totalFiles int, completed bool, err error) {
	globusTask, err := client.TransferGetTaskByID(globusTaskId)
	if err != nil {
		return 0, 0, 1, false, fmt.Errorf("globus: can't continue transfer because an error occured while polling the task \"%s\": %v", globusTaskId, err)
	}
	switch globusTask.Status {
	case "ACTIVE":
		totalFiles := globusTask.Files
		if globusTask.FilesSkipped != nil {
			totalFiles -= *globusTask.FilesSkipped
		}
		return globusTask.BytesTransferred, globusTask.FilesTransferred, totalFiles, false, nil
	case "INACTIVE":
		return 0, 0, 1, false, fmt.Errorf("globus: transfer became inactive, manual intervention required")
	case "SUCCEEDED":
		totalFiles := globusTask.Files
		if globusTask.FilesSkipped != nil {
			totalFiles -= *globusTask.FilesSkipped
		}
		return globusTask.BytesTransferred, globusTask.FilesTransferred, totalFiles, true, nil
	case "FAILED":
		return 0, 0, 1, false, fmt.Errorf("globus: task failed with the following error - code: \"%s\" description: \"%s\"", globusTask.FatalError.Code, globusTask.FatalError.Description)
	default:
		return 0, 0, 1, false, fmt.Errorf("globus: unknown task status: %s", globusTask.Status)
	}
}

func GlobusTransfer(globusConf task.GlobusTransferConfig, t *task.TransferTask, taskCtx context.Context, localTaskId uuid.UUID, datasetFolder string, fileList []datasetIngestor.Datafile, notifier task.ProgressNotifier) error {
	client, ok := t.GetTransferObject("globus_client").(*globus.GlobusClient)
	if !ok {
		return fmt.Errorf("globus client is not set for this task")
	}

	// transfer given filelist
	var filePathList []string
	var fileIsSymlinkList []bool
	for _, file := range fileList {
		filePathList = append(filePathList, filepath.ToSlash(file.Path))
		fileIsSymlinkList = append(fileIsSymlinkList, file.IsSymlink)
	}
	datasetFolder = filepath.ToSlash(datasetFolder)

	s := strings.Split(strings.Trim(datasetFolder, "/"), "/")
	datasetFolderName := s[len(s)-1]

	result, err := client.TransferFileList(
		globusConf.SourceCollection,
		globusConf.SourcePrefixPath+"/"+datasetFolder,
		globusConf.DestinationCollection,
		globusConf.DestinationPrefixPath+"/"+datasetFolderName,
		filePathList,
		fileIsSymlinkList,
		true,
	)
	if err != nil {
		return fmt.Errorf("globus: an error occured when requesting dataset transfer: %v", err)
	}
	if result.Code != "Accepted" {
		return fmt.Errorf("globus: transfer was not accepted - code: \"%s\", message: \"%s\"", result.Code, result.Message)
	}

	// task monitoring
	globusTaskId := result.TaskId
	var taskCompleted bool
	var bytesTransferred, filesTransferred, totalFiles int

	// note: the totalFiles variable here uses the count returned by Globus
	//   this can change over the course of the transfer, as Globus succeeds in finding the files
	//   (recursion, checking their existence...)

	bytesTransferred, filesTransferred, totalFiles, taskCompleted, err = globusCheckTransfer(client, globusTaskId)
	if err != nil {
		return err
	}
	if totalFiles == 0 {
		totalFiles = 1 // needed because percentage meter goes NaN otherwise
	}
	status := task.Transferring
	if taskCompleted {
		status = task.Finished
	}
	t.UpdateDetails(
		task.SetBytesTransferred(bytesTransferred),
		task.SetFilesTransferred(filesTransferred),
		task.SetFilesTotal(totalFiles),
		task.SetStatus(status),
		task.SetMessage("transfering"),
	)
	if taskCompleted {
		return nil
	}
	notifier.OnTaskProgress(localTaskId, 100*filesTransferred/totalFiles)

	transferUpdater := time.After(1 * time.Minute)
	for {
		select {
		case <-taskCtx.Done():
			// we're cancelling the task
			result, err := client.TransferCancelTaskByID(globusTaskId)
			if err != nil {
				return fmt.Errorf("globus: couldn't cancel task: %v", err)
			}
			if result.Code != "Canceled" {
				return fmt.Errorf("globus: couldn't cancel task - code: \"%s\", message: \"%s\"", result.Code, result.Message)
			}
			t.UpdateDetails(
				task.SetMessage("task was cancelled"),
				task.SetStatus(task.Cancelled),
			)
			notifier.OnTaskCanceled(localTaskId)
			return nil
		case <-transferUpdater:
			// check state of transfer
			transferUpdater = time.After(1 * time.Minute)
			bytesTransferred, filesTransferred, totalFiles, taskCompleted, err = globusCheckTransfer(client, globusTaskId)
			if err != nil {
				return err // transfer cannot be finished: irrecoverable error
			}
			if totalFiles == 0 {
				totalFiles = 1 // needed because percentage meter goes NaN otherwise
			}

			t.UpdateDetails(task.SetBytesTransferred(bytesTransferred), task.SetFilesTransferred(filesTransferred))
			notifier.OnTaskProgress(localTaskId, 100*filesTransferred/totalFiles)

			if taskCompleted {
				return nil // we're done!
			}
		}
	}
}
