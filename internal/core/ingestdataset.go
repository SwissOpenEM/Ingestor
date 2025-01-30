package core

import (
	"bufio"
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/SwissOpenEM/Ingestor/internal/task"
	"github.com/fatih/color"
	"github.com/paulscherrerinstitute/scicat-cli/v3/datasetIngestor"
	"github.com/paulscherrerinstitute/scicat-cli/v3/datasetUtils"
)

const MAX_FILES = 400000

func createLocalSymlinkCallbackForFileLister(skipSymlinks *string, skippedLinks *uint) func(symlinkPath string, sourceFolder string) (bool, error) {
	scanner := bufio.NewScanner(os.Stdin)
	return func(symlinkPath string, sourceFolder string) (bool, error) {
		var keep bool
		pointee, _ := os.Readlink(symlinkPath) // just pass the file name
		if !filepath.IsAbs(pointee) {
			symlinkAbs, err := filepath.Abs(filepath.Dir(symlinkPath))
			if err != nil {
				return false, err
			}
			// log.Printf(" CWD path pointee :%v %v %v", dir, filepath.Dir(path), pointee)
			pointeeAbs := filepath.Join(symlinkAbs, pointee)
			pointee, err = filepath.EvalSymlinks(pointeeAbs)
			if err != nil {
				log.Printf("Could not follow symlink for file:%v %v", pointeeAbs, err)
				keep = false
				log.Printf("keep variable set to %v", keep)
			}
		}
		//fmt.Printf("Skip variable:%v\n", *skip)
		if *skipSymlinks == "ka" || *skipSymlinks == "kA" {
			keep = true
		} else if *skipSymlinks == "sa" || *skipSymlinks == "sA" {
			keep = false
		} else if *skipSymlinks == "da" || *skipSymlinks == "dA" {
			keep = strings.HasPrefix(pointee, sourceFolder)
		} else {
			color.Set(color.FgYellow)
			log.Printf("Warning: the file %s is a link pointing to %v.", symlinkPath, pointee)
			color.Unset()
			log.Printf(`
	Please test if this link is meaningful and not pointing 
	outside the sourceFolder %s. The default behaviour is to
	keep only internal links within a source folder.
	You can also specify that you want to apply the same answer to ALL 
	subsequent links within the current dataset, by appending an a (dA,ka,sa).
	If you want to give the same answer even to all subsequent datasets 
	in this command then specify a capital 'A', e.g. (dA,kA,sA)
	Do you want to keep the link in dataset or skip it (D(efault)/k(eep)/s(kip) ?`, sourceFolder)
			scanner.Scan()
			*skipSymlinks = scanner.Text()
			if *skipSymlinks == "" {
				*skipSymlinks = "d"
			}
			if *skipSymlinks == "d" || *skipSymlinks == "dA" {
				keep = strings.HasPrefix(pointee, sourceFolder)
			} else {
				keep = (*skipSymlinks != "s" && *skipSymlinks != "sa" && *skipSymlinks != "sA")
			}
		}
		if keep {
			color.Set(color.FgGreen)
			log.Printf("You chose to keep the link %v -> %v.\n\n", symlinkPath, pointee)
		} else {
			color.Set(color.FgRed)
			*skippedLinks++
			log.Printf("You chose to remove the link %v -> %v.\n\n", symlinkPath, pointee)
		}
		color.Unset()
		return keep, nil
	}
}

func createLocalFilenameFilterCallback(illegalFileNamesCounter *uint) func(filepath string) bool {
	return func(filepath string) (keep bool) {
		keep = true
		// make sure that filenames do not contain characters like "\" or "*"
		if strings.ContainsAny(filepath, "*\\") {
			color.Set(color.FgRed)
			log.Printf("Warning: the file %s contains illegal characters like *,\\ and will not be archived.", filepath)
			color.Unset()
			if illegalFileNamesCounter != nil {
				*illegalFileNamesCounter++
			}
			keep = false
		}
		// and check for triple blanks, they are used to separate columns in messages
		if keep && strings.Contains(filepath, "   ") {
			color.Set(color.FgRed)
			log.Printf("Warning: the file %s contains 3 consecutive blanks which is not allowed. The file not be archived.", filepath)
			color.Unset()
			if illegalFileNamesCounter != nil {
				*illegalFileNamesCounter++
			}
			keep = false
		}
		return keep
	}
}

func CheckIfFolderExists(path string) error {
	// check if the folder exists
	fileInfo, err := os.Stat(path)
	if err != nil {
		return err
	}
	if !fileInfo.IsDir() {
		return errors.New("'sourceFolder' is not a directory")
	}
	return nil
}

func AddDatasetToScicat(
	metaDataMap map[string]interface{},
	folderPath string,
	userToken string,
	scicatUrl string,
) (datasetId string, totalSize int64, fileList []datasetIngestor.Datafile, err error) {
	var http_client = &http.Client{
		Transport: &http.Transport{TLSClientConfig: &tls.Config{InsecureSkipVerify: true}},
		Timeout:   120 * time.Second}

	SCICAT_API_URL := scicatUrl

	const TAPECOPIES = 2 // dummy value, unused
	const DATASETFILELISTTXT = ""

	var skipSymlinks string = "dA" // skip all simlinks

	user := map[string]string{
		"accessToken": userToken,
	}

	datasetFolder := folderPath

	fullUser, accessGroups, err := datasetUtils.GetUserInfoFromToken(http_client, SCICAT_API_URL, userToken)
	if err != nil {
		return datasetId, totalSize, fileList, err
	}

	// check if dataset already exists (identified by source folder)
	_, _, err = datasetIngestor.CheckMetadata(http_client, SCICAT_API_URL, metaDataMap, fullUser, accessGroups)
	if err != nil {
		return datasetId, totalSize, fileList, err
	}

	var skippedLinks uint = 0
	var illegalFileNames uint = 0
	localSymlinkCallback := createLocalSymlinkCallbackForFileLister(&skipSymlinks, &skippedLinks)
	localFilepathFilterCallback := createLocalFilenameFilterCallback(&illegalFileNames)

	// collect (local) files
	fileList, startTime, endTime, owner, numFiles, totalSize, err := datasetIngestor.GetLocalFileList(datasetFolder, DATASETFILELISTTXT, localSymlinkCallback, localFilepathFilterCallback)
	if err != nil {
		log.Printf("")
		return datasetId, totalSize, fileList, err
	}

	// size & filecount checks
	if totalSize == 0 {
		return datasetId, totalSize, fileList, errors.New("can't ingest: the total size of the dataset is 0")
	}
	if numFiles > MAX_FILES {
		return datasetId, totalSize, fileList, fmt.Errorf("can't ingest: the number of files (%d) exceeds the max. allowed (%d)", numFiles, MAX_FILES)
	}

	originalMetaDataMap := map[string]string{}
	datasetIngestor.UpdateMetaData(http_client, SCICAT_API_URL, user, originalMetaDataMap, metaDataMap, startTime, endTime, owner, TAPECOPIES)

	metaDataMap["datasetlifecycle"] = map[string]interface{}{}
	metaDataMap["datasetlifecycle"].(map[string]interface{})["isOnCentralDisk"] = false
	metaDataMap["datasetlifecycle"].(map[string]interface{})["archiveStatusMessage"] = "filesNotYetAvailable"
	metaDataMap["datasetlifecycle"].(map[string]interface{})["archivable"] = false

	// NOTE: scicat-cli considers "ingestion" as just inserting the dataset into scicat and adding the orig datablocks
	datasetId, err = datasetIngestor.IngestDataset(http_client, SCICAT_API_URL, metaDataMap, fileList, user)

	// TODO: add attachments here if it's going to be needed

	return datasetId, totalSize, fileList, err
}

func TransferDataset(
	task_context context.Context,
	it *task.TransferTask,
	serviceUser *UserCreds,
	config Config,
	notifier ProgressNotifier,
) error {
	datasetId := it.GetDatasetId()
	datasetFolder := it.DatasetFolder.FolderPath
	fileList := it.GetFileList()
	var err error
	var http_client = &http.Client{
		Transport: &http.Transport{TLSClientConfig: &tls.Config{InsecureSkipVerify: true}},
		Timeout:   120 * time.Second}

	switch it.TransferMethod {
	case task.TransferS3:
		_, err = UploadS3(task_context, datasetId, datasetFolder, it.DatasetFolder.Id, config.Transfer.S3, notifier)
	case task.TransferGlobus:
		// globus doesn't work with absolute folders, this library uses sourcePrefix to adapt the path to the globus' own path from a relative path
		relativeDatasetFolder := strings.TrimPrefix(datasetFolder, config.WebServer.CollectionLocation)
		err = GlobusTransfer(config.Transfer.Globus, it, task_context, it.DatasetFolder.Id, relativeDatasetFolder, fileList, notifier)
	_:
	}

	if err != nil {
		return err
	}

	// mark dataset archivable
	if serviceUser == nil {
		return fmt.Errorf("no service user was set, can't mark dataset as archivable")
	}
	user, _, err := datasetUtils.AuthenticateUser(http_client, config.Scicat.Host, serviceUser.Username, serviceUser.Password, false)
	if err != nil {
		return err
	}
	err = datasetIngestor.MarkFilesReady(http_client, config.Scicat.Host, datasetId, user)
	return err
}
