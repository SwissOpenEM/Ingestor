package core

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/SwissOpenEM/globus"
	"github.com/google/uuid"
	"golang.org/x/oauth2"
)

var globusClient globus.GlobusClient

func GlobusCliLogIn(gConfig *GlobusTransferConfig) error {
	// config setup
	ctx := context.Background()
	clientConfig := globus.AuthGenerateOauthClientConfig(ctx, gConfig.ClientID, gConfig.ClientSecret, gConfig.RedirectURL, gConfig.Scopes)
	verifier := oauth2.GenerateVerifier()
	clientConfig.AuthCodeURL("state", oauth2.AccessTypeOffline, oauth2.S256ChallengeOption(verifier))

	// redirect user to consent page to ask for permission and obtain the code
	url := clientConfig.AuthCodeURL("state", oauth2.AccessTypeOffline, oauth2.S256ChallengeOption(verifier))
	fmt.Printf("Visit the URL for the auth dialog: %v\n\nEnter the received code here: ", url)

	// get token
	var code string
	if _, err := fmt.Scan(&code); err != nil {
		return err
	}
	tok, err := clientConfig.Exchange(ctx, code, oauth2.VerifierOption(verifier))
	if err != nil {
		return fmt.Errorf("oauth2 exchange failed: %v", err)
	}

	// setup auto renewal of tokens, update refresh token field in config and create client
	ts := clientConfig.TokenSource(ctx, tok)
	client := oauth2.NewClient(ctx, ts)

	gConfig.RefreshToken = tok.RefreshToken

	// set globus client and return
	GlobusSetHttpClient(client)
	return nil
}

func GlobusLoginWithRefreshToken(gConfig GlobusTransferConfig) {
	ctx := context.Background()
	clientConfig := globus.AuthGenerateOauthClientConfig(ctx, gConfig.ClientID, gConfig.ClientSecret, gConfig.RedirectURL, gConfig.Scopes)
	tok := oauth2.Token{
		RefreshToken: gConfig.RefreshToken,
		Expiry:       time.Date(1950, 1, 1, 0, 0, 0, 0, time.UTC),
		AccessToken:  "",
		TokenType:    "Bearer",
	}
	ts := clientConfig.TokenSource(ctx, &tok)
	client := oauth2.NewClient(ctx, ts)
	GlobusSetHttpClient(client)
}

func GlobusIsClientReady() bool {
	return globusClient.IsClientSet()
}

func GlobusSetHttpClient(client *http.Client) {
	globusClient = globus.HttpClientToGlobusClient(client)
}

func globusCheckTransfer(globusTaskId string, localTaskId uuid.UUID) (filesTransferred int, totalFiles int, completed bool, err error) {
	task, err := globusClient.TransferGetTaskByID(globusTaskId)
	if err != nil {
		return 0, 1, false, fmt.Errorf("globus: can't continue transfer because an error occured while polling the task \"%s\": %v", globusTaskId, err)
	}
	switch task.Status {
	case "ACTIVE":
		totalFiles := task.Files
		if task.FilesSkipped != nil {
			totalFiles -= *task.FilesSkipped
		}
		return task.FilesTransferred, totalFiles, false, nil
	case "INACTIVE":
		return 0, 1, false, fmt.Errorf("globus: transfer became inactive, manual intervention required")
	case "SUCCEEDED":
		totalFiles := task.Files
		if task.FilesSkipped != nil {
			totalFiles -= *task.FilesSkipped
		}
		return task.FilesTransferred, totalFiles, true, nil
	case "FAILED":
		return 0, 1, false, fmt.Errorf("globus: task failed with the following error - code: \"%s\" description: \"%s\"", task.FatalError.Code, task.FatalError.Description)
	default:
		return 0, 1, false, fmt.Errorf("globus: unknown task status: %s", task.Status)
	}
}

func GlobusTransfer(globusConf GlobusTransferConfig, taskCtx context.Context, localTaskId uuid.UUID, datasetFolder string, notifier ProgressNotifier) error {
	// check if globus client is properly set up, use refresh token if available
	if !globusClient.IsClientSet() {
		if globusConf.RefreshToken == "" {
			return fmt.Errorf("globus: not logged into globus")
		}
		GlobusLoginWithRefreshToken(globusConf)
	}

	// for now, we're using recursive folder sync of globus, it does not handle symlinks how we want however
	// TODO: use TransferFileList (but potentially it'll still not handle symlinks how we want it to...)
	result, err := globusClient.TransferFolderSync(
		globusConf.SourceCollection,
		globusConf.SourcePrefixPath+"/"+datasetFolder,
		globusConf.DestinationCollection,
		globusConf.DestinationPrefixPath+"/"+datasetFolder,
		true,
	)
	if err != nil {
		return fmt.Errorf("globus: an error occured when requesting dataset transfer: %v", err)
	}
	if result.Code != "Accepted" {
		return fmt.Errorf("globus: transfer was not accepted - code: \"%s\", message: \"%s\"", result.Code, result.Message)
	}

	globusTaskId := result.TaskId

	// start periodically checking transfer until done, failed or cancelled
	startTime := time.Now()

	var taskCompleted bool
	var filesTransferred, totalFiles int
	filesTransferred, totalFiles, taskCompleted, err = globusCheckTransfer(globusTaskId, localTaskId)
	if err != nil {
		return err
	}
	if taskCompleted {
		return nil
	}
	if totalFiles == 0 {
		totalFiles = 1 // needed because percentage meter goes NaN otherwise
	}
	notifier.OnTaskProgress(localTaskId, filesTransferred, totalFiles, int(time.Since(startTime).Seconds()))
	timerUpdater := time.After(1 * time.Second)
	transferUpdater := time.After(1 * time.Minute)
	for {
		select {
		case <-taskCtx.Done():
			// we're cancelling the task
			result, err := globusClient.TransferCancelTaskByID(globusTaskId)
			if err != nil {
				return fmt.Errorf("globus: couldn't cancel task: %v", err)
			}
			if result.Code != "Canceled" {
				return fmt.Errorf("globus: couldn't cancel task - code: \"%s\", message: \"%s\"", result.Code, result.Message)
			}
			notifier.OnTaskCanceled(localTaskId)
			return nil
		case <-timerUpdater:
			// update timer every second
			timerUpdater = time.After(1 * time.Second)
			notifier.OnTaskProgress(localTaskId, filesTransferred, totalFiles, int(time.Since(startTime).Seconds()))
		case <-transferUpdater:
			// check state of transfer
			transferUpdater = time.After(1 * time.Minute)
			filesTransferred, totalFiles, taskCompleted, err = globusCheckTransfer(globusTaskId, localTaskId)
			if err != nil {
				return err // transfer cannot be finished: irrecoverable error
			}
			if totalFiles == 0 {
				totalFiles = 1 // needed because percentage meter goes NaN otherwise
			}
			notifier.OnTaskProgress(localTaskId, filesTransferred, totalFiles, int(time.Since(startTime).Seconds()))
			if taskCompleted {
				return nil // we're done!
			}
		}
	}
}
