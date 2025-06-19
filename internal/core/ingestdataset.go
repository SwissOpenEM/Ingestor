package core

import (
	"bufio"
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/SwissOpenEM/Ingestor/internal/globustransfer"
	"github.com/SwissOpenEM/Ingestor/internal/s3upload"
	"github.com/SwissOpenEM/Ingestor/internal/transfertask"
	"github.com/SwissOpenEM/globus"
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
			// log().Printf(" CWD path pointee :%v %v %v", dir, filepath.Dir(path), pointee)
			pointeeAbs := filepath.Join(symlinkAbs, pointee)
			pointee, err = filepath.EvalSymlinks(pointeeAbs)
			if err != nil {
				log().Error("Could not follow symlink for file:%v %v", pointeeAbs, err)
				keep = false
				log().Info(fmt.Sprintf("keep variable set to %t", keep))
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
			log().Warn(fmt.Sprintf("The file %s is a link pointing to %v.", symlinkPath, pointee))
			log().Warn(fmt.Sprintf(`
	Please test if this link is meaningful and not pointing 
	outside the sourceFolder %s. The default behaviour is to
	keep only internal links within a source folder.
	You can also specify that you want to apply the same answer to ALL 
	subsequent links within the current dataset, by appending an a (dA,ka,sa).
	If you want to give the same answer even to all subsequent datasets 
	in this command then specify a capital 'A', e.g. (dA,kA,sA)
	Do you want to keep the link in dataset or skip it (D(efault)/k(eep)/s(kip) ?`, sourceFolder))
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
			log().Info("You chose to keep the link %v -> %v.\n\n", symlinkPath, pointee)
		} else {
			*skippedLinks++
			log().Warn(fmt.Sprintf("You chose to remove the link %v -> %v.\n\n", symlinkPath, pointee))
		}
		return keep, nil
	}
}

func createLocalFilenameFilterCallback(illegalFileNamesCounter *uint) func(filepath string) bool {
	return func(filepath string) (keep bool) {
		keep = true
		// make sure that filenames do not contain characters like "\" or "*"
		if strings.ContainsAny(filepath, "*\\") {
			log().Warn(fmt.Sprintf("The file %s contains illegal characters like *,\\ and will not be archived.", filepath))
			if illegalFileNamesCounter != nil {
				*illegalFileNamesCounter++
			}
			keep = false
		}
		// and check for triple blanks, they are used to separate columns in messages
		if keep && strings.Contains(filepath, "   ") {
			log().Warn(fmt.Sprintf("The file %s contains 3 consecutive blanks which is not allowed. The file not be archived.", filepath))
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

const (
	ErrIllegalKeys = "metadata contains keys with illegal characters (., [], $, or <>)"
)

func AddDatasetToScicat(
	metaDataMap map[string]interface{},
	datasetFolder string,
	storageLocation string,
	userToken string,
	scicatUrl string,
	isOnCentralDisk bool,
) (datasetId string, totalSize int64, fileList []datasetIngestor.Datafile, username string, err error) {
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

	fullUser, accessGroups, err := datasetUtils.GetUserInfoFromToken(http_client, SCICAT_API_URL, userToken)
	if err != nil {
		return datasetId, totalSize, fileList, "", err
	}

	if keys := datasetIngestor.CollectIllegalKeys(metaDataMap); len(keys) > 0 {
		return datasetId, totalSize, fileList, "", errors.New(ErrIllegalKeys + ": \"" + strings.Join(keys, "\", \"") + "\"")
	}

	_, err = datasetIngestor.CheckUserAndOwnerGroup(user, accessGroups, metaDataMap)
	if err != nil {
		return datasetId, totalSize, fileList, "", err
	}

	err = datasetIngestor.CheckMetadataValidity(http_client, SCICAT_API_URL, user["accessToken"], metaDataMap)
	if err != nil {
		return datasetId, totalSize, fileList, "", err
	}

	var skippedLinks uint = 0
	var illegalFileNames uint = 0
	localSymlinkCallback := createLocalSymlinkCallbackForFileLister(&skipSymlinks, &skippedLinks)
	localFilepathFilterCallback := createLocalFilenameFilterCallback(&illegalFileNames)

	// collect (local) files
	fileList, startTime, endTime, owner, numFiles, totalSize, err := datasetIngestor.GetLocalFileList(datasetFolder, DATASETFILELISTTXT, localSymlinkCallback, localFilepathFilterCallback)
	if err != nil {
		return datasetId, totalSize, fileList, "", err
	}

	// size & filecount checks
	if totalSize == 0 {
		return datasetId, totalSize, fileList, "", errors.New("can't ingest: the total size of the dataset is 0")
	}
	if numFiles > MAX_FILES {
		return datasetId, totalSize, fileList, "", fmt.Errorf("can't ingest: the number of files (%d) exceeds the max. allowed (%d)", numFiles, MAX_FILES)
	}

	originalMetaDataMap := map[string]string{}
	datasetIngestor.UpdateMetaData(http_client, SCICAT_API_URL, user, originalMetaDataMap, metaDataMap, startTime, endTime, owner, TAPECOPIES)

	metaDataMap["datasetlifecycle"] = map[string]interface{}{}
	metaDataMap["datasetlifecycle"].(map[string]interface{})["isOnCentralDisk"] = isOnCentralDisk
	if isOnCentralDisk {
		metaDataMap["datasetlifecycle"].(map[string]interface{})["archiveStatusMessage"] = "filesReady"
	} else {
		metaDataMap["datasetlifecycle"].(map[string]interface{})["archiveStatusMessage"] = "filesNotYetAvailable"
	}
	metaDataMap["datasetlifecycle"].(map[string]interface{})["archivable"] = isOnCentralDisk
	metaDataMap["datasetlifecycle"].(map[string]interface{})["storageLocation"] = storageLocation

	// NOTE: scicat-cli considers "ingestion" as just inserting the dataset into scicat and adding the orig datablocks
	datasetId, err = datasetIngestor.IngestDataset(http_client, SCICAT_API_URL, metaDataMap, fileList, user)

	// TODO: add attachments here if it's going to be needed

	return datasetId, totalSize, fileList, fullUser["username"], err
}

func TransferDataset(
	task_context context.Context,
	transferTask *transfertask.TransferTask,
	serviceUser *UserCreds,
	config Config,
	notifier transfertask.ProgressNotifier,
) error {

	var err error

	switch transferTask.TransferMethod {
	case transfertask.TransferS3:
		accessToken, ok := transferTask.GetTransferObject("accessToken").(string)
		if !ok {
			return fmt.Errorf("missing access token for s3 upload")
		}
		refreshToken, ok := transferTask.GetTransferObject("refreshToken").(string)
		if !ok {
			return fmt.Errorf("missing refresh token for s3 upload")
		}
		expires_in, ok := transferTask.GetTransferObject("expires_in").(int)
		if !ok {
			return fmt.Errorf("missing expiration for s3 upload token")
		}

		tokenSource := s3upload.CreateTokenSource(context.Background(), config.Transfer.S3.ClientID, config.Transfer.S3.TokenUrl, accessToken, refreshToken, expires_in)

		err = s3upload.UploadS3(task_context, transferTask, config.Transfer.S3, tokenSource, notifier)

		if err == nil {
			archivalJobInfo := transferTask.GetArchivalJobInfo()
			err = s3upload.FinalizeUpload(task_context, config.Transfer.S3, transferTask.GetDatasetId(), archivalJobInfo.OwnerUser, archivalJobInfo.OwnerGroup, archivalJobInfo.ContactEmail, archivalJobInfo.AutoArchive, tokenSource)
		}

	case transfertask.TransferGlobus:
		// get transfer objects
		datasetFolder := transferTask.DatasetFolder.FolderPath
		fileList := transferTask.GetFileList()
		client, ok := transferTask.GetTransferObject("globus_client").(*globus.GlobusClient)
		if !ok {
			return fmt.Errorf("globus client was not set")
		}
		username, ok := transferTask.GetTransferObject("username").(string)
		if !ok {
			return fmt.Errorf("username was not set for globus transfer")
		}

		files := make([]globustransfer.File, len(fileList))
		bytesTotal := int64(0)
		for i, file := range fileList {
			files[i].IsSymlink = file.IsSymlink
			files[i].Path = file.Path
			bytesTotal += int64(file.Size)
		}

		transferNotifier := transfertask.NewTransferNotifier(bytesTotal, transferTask.DatasetFolder.Id, notifier, transferTask)

		transferTask.TransferStarted()
		err = globustransfer.TransferFiles(
			client,
			config.Transfer.Globus.SourceCollectionID,
			config.Transfer.Globus.CollectionRootPath,
			config.Transfer.Globus.DestinationCollectionID,
			config.Transfer.Globus.DestinationTemplate,
			transferTask.GetDatasetId(),
			username,
			task_context,
			datasetFolder,
			files,
			&transferNotifier,
		)
		if err != nil {
			return err
		}

		err = FinalizeTransfer(serviceUser, config, transferTask.GetDatasetId(), transferTask.GetArchivalJobInfo())
		if err != nil {
			return err
		}

	default:
		err = fmt.Errorf("unknown transfer method: %d", transferTask.TransferMethod)
	}

	return err
}

func FinalizeTransfer(serviceUser *UserCreds, config Config, datasetId string, archivalJobInfo transfertask.ArchivalJobInfo) error {
	var http_client = &http.Client{
		Transport: &http.Transport{TLSClientConfig: &tls.Config{InsecureSkipVerify: true}},
		Timeout:   120 * time.Second}
	// mark dataset archivable
	if serviceUser == nil {
		return fmt.Errorf("no service user was set, can't mark dataset as archivable")
	}
	user, _, err := datasetUtils.AuthenticateUser(http_client, config.Scicat.Host, serviceUser.Username, serviceUser.Password, false)
	if err != nil {
		return err
	}
	err = datasetIngestor.MarkFilesReady(http_client, config.Scicat.Host, datasetId, user)
	if err != nil {
		return err
	}

	// Create the archiving job as the user that is logged in and not the the service user
	user["mail"] = archivalJobInfo.ContactEmail
	user["username"] = archivalJobInfo.OwnerUser

	// auto archive
	if archivalJobInfo.AutoArchive {
		copies := 1
		_, err = datasetUtils.CreateArchivalJob(http_client, config.Scicat.Host, user, archivalJobInfo.OwnerGroup, []string{datasetId}, &copies)
	}
	return err
}
