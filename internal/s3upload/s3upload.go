package s3upload

import (
	"context"
	"fmt"
	"os"
	"path"
	"time"

	"github.com/SwissOpenEM/Ingestor/internal/transfertask"
	"github.com/google/uuid"
	openapi_types "github.com/oapi-codegen/runtime/types"

	"golang.org/x/oauth2"
	"golang.org/x/sync/errgroup"
)

const (
	GiB = (1024 * 1024 * 1024)
)

type S3Objects struct {
	Files       []string
	ObjectNames []string
	TotalBytes  int64
}

func GetTokens(ctx context.Context, endpoint string, userToken string) (string, string, int, error) {
	resp, err := GetPresignedURLServer(endpoint).CreateNewServiceTokenWithResponse(ctx,
		createAddAuthorizationHeaderFunction(userToken))

	if err != nil {
		return "", "", 0, err
	}

	if resp.HTTPResponse.StatusCode != 201 {
		return "", "", 0, fmt.Errorf("failed to get access tokens: %d, %s", resp.HTTPResponse.StatusCode, resp.HTTPResponse.Status)
	}

	return resp.JSON201.AccessToken, resp.JSON201.RefreshToken, *resp.JSON201.ExpiresIn, nil
}

func CreateTokenSource(ctx context.Context, clientID string, tokenURL string, accessToken string, refreshToken string, expiresIn int) oauth2.TokenSource {
	config := &oauth2.Config{
		ClientID: clientID,
		Endpoint: oauth2.Endpoint{TokenURL: tokenURL},
		// required for the refresh token to be updated
		Scopes: []string{"offline_access"},
	}

	token := &oauth2.Token{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		Expiry:       time.Now().Add(time.Duration(expiresIn)),
	}

	return config.TokenSource(ctx, token)
}

// Upload all files in a folder using presinged urls
func UploadS3(ctx context.Context, task *transfertask.TransferTask, options transfertask.S3TransferConfig, tokenSource oauth2.TokenSource, notifier transfertask.ProgressNotifier) error {

	if len(task.GetFileList()) == 0 {
		return fmt.Errorf("empty file list provided")
	}

	datasetFolder := task.DatasetFolder.FolderPath
	datasetID := task.GetDatasetID()
	uploadID := task.DatasetFolder.ID

	s3Objects := S3Objects{}
	for _, f := range task.GetFileList() {
		info, _ := os.Stat(path.Join(datasetFolder, f.Path))
		if info.IsDir() {
			continue
		}
		s3Objects.TotalBytes += info.Size()
		s3Objects.Files = append(s3Objects.Files, path.Join(datasetFolder, f.Path))
		s3Objects.ObjectNames = append(s3Objects.ObjectNames, "openem-network/datasets/"+datasetID+"/raw_files/"+f.Path)
	}

	transferNotifier := transfertask.NewTransferNotifier(s3Objects.TotalBytes, uploadID, notifier, task)

	var canUpload = false

	// retry every 10 minutes and then every hour
	nextInterval := func(i int) time.Duration {
		if i < 10 {
			return 10 * time.Minute
		} else {
			return 1 * time.Hour
		}
	}

	retry := 0
	for !canUpload {
		token, err := tokenSource.Token()
		if err != nil {
			continue
		}
		// query api server
		resp, err := GetPresignedURLServer(options.Endpoint).RequestDatasetUploadWithResponse(ctx, UploadRequestBody{
			DatasetId:      datasetID,
			DatasetSizeGib: int(s3Objects.TotalBytes) / GiB,
		}, createAddAuthorizationHeaderFunction(token.AccessToken))

		if err != nil {
			return err
		}

		if resp.JSON201 != nil {
			canUpload = true
		} else if resp.JSON507 != nil {
			next := nextInterval(retry)
			retry += 1
			log().Info(fmt.Sprintf("Could not start upload at this point in time. Waiting %f s", next.Seconds()))
			time.Sleep(next)
		} else {
			return fmt.Errorf("RequestDatasetUpload request failed: %w", err)
		}
	}

	task.TransferStarted()

	return uploadFiles(ctx, &s3Objects, options, &transferNotifier, uploadID, tokenSource)
}

func uploadFiles(ctx context.Context, s3Objects *S3Objects, options transfertask.S3TransferConfig, transferNotifier *transfertask.TransferNotifier, uploadID uuid.UUID, tokenSource oauth2.TokenSource) error {
	errorGroup, context := errgroup.WithContext(ctx)
	objectsChannel := make(chan int, len(s3Objects.Files))

	nWorkers := min(options.ConcurrentFiles, len(s3Objects.Files))

	for t := 0; t < nWorkers; t++ {
		errorGroup.Go(
			func() error {
				for idx := range objectsChannel {
					select {
					case <-context.Done():
						transferNotifier.OnTaskCanceled(uploadID)
						return context.Err()
					default:
						err := uploadFile(context, s3Objects.Files[idx], s3Objects.ObjectNames[idx], options, transferNotifier, tokenSource)
						if err != nil {
							return err
						}
					}
				}
				return nil
			})
	}
	for idx := range s3Objects.Files {
		objectsChannel <- idx
	}
	close(objectsChannel)
	return errorGroup.Wait()

}

func FinalizeUpload(ctx context.Context, config transfertask.S3TransferConfig, datasetPID string, ownerUser string, ownerGroup string, email string, autoArchive bool, tokenSource oauth2.TokenSource) error {

	token, err := tokenSource.Token()
	if err != nil {
		return fmt.Errorf("finalizing upload failed: error fetching a new token: %w", err)
	}

	resp, err := GetPresignedURLServer(config.Endpoint).FinalizeDatasetUploadWithResponse(ctx, FinalizeDatasetUploadBody{
		DatasetPid:         datasetPID,
		OwnerUser:          ownerUser,
		OwnerGroup:         ownerGroup,
		ContactEmail:       openapi_types.Email(email),
		CreateArchivingJob: autoArchive,
	}, createAddAuthorizationHeaderFunction(token.AccessToken))

	if err != nil {
		return err
	}

	switch resp.HTTPResponse.StatusCode {
	case 500:
		return fmt.Errorf("failed to finalize upload: %d, %s, %s ", resp.HTTPResponse.StatusCode, resp.HTTPResponse.Status, *resp.JSON500.Details)
	case 201:
		log().Debug("Upload finalized", "dataset pid", resp.JSON201.DatasetId, "message", resp.JSON201.Message)
	default:
		return fmt.Errorf("failed to finalize upload: %d, %s", resp.HTTPResponse.StatusCode, resp.HTTPResponse.Status)
	}

	return nil
}
