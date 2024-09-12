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
		Expiry:       time.Date(1950, 1, 1, 0, 0, 0, 0, nil),
		AccessToken:  "",
		TokenType:    "Bearer",
	}
	ts := clientConfig.TokenSource(ctx, &tok)
	client := oauth2.NewClient(ctx, ts)
	GlobusSetHttpClient(client)
}

func GlobusSetHttpClient(client *http.Client) {
	globusClient = globus.HttpClientToGlobusClient(client)
}

func globusCheckTransfer(globusTaskId string, localTaskId uuid.UUID, notifier ProgressNotifier) (completed bool, err error) {
	task, err := globusClient.TransferGetTaskByID(globusTaskId)
	if err != nil {
		return false, fmt.Errorf("globus: can't continue transfer because an error occured while polling the task \"%s\": %v", globusTaskId, err)
	}
	switch task.Status {
	case "ACTIVE":
		totalFiles := task.Files
		if task.FilesSkipped != nil {
			totalFiles -= *task.FilesSkipped
		}
		notifier.OnTaskProgress(localTaskId, task.FilesTransferred, totalFiles, 0)
		return false, nil
	case "INACTIVE":
		return false, fmt.Errorf("globus: transfer became inactive, manual intervention required")
	case "SUCCEEDED":
		notifier.OnTaskCompleted(localTaskId, 0)
		return true, nil
	case "FAILED":
		return false, fmt.Errorf("globus: task failed with the following error - code: \"%s\" description: \"%s\"", task.FatalError.Code, task.FatalError.Description)
	default:
		return false, fmt.Errorf("globus: unknown task status: %s", task.Status)
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

	// periodically check transfer until done, failed or cancelled
	var taskCompleted bool
	taskCompleted, err = globusCheckTransfer(globusTaskId, localTaskId, notifier)
	if err != nil {
		return err
	}
	if taskCompleted {
		return nil
	}
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
		case <-time.After(1 * time.Minute):
			// check state of transfer
			taskCompleted, err = globusCheckTransfer(globusTaskId, localTaskId, notifier)
			if err != nil {
				return err
			}
			if taskCompleted {
				return nil
			}
		}
	}
}
