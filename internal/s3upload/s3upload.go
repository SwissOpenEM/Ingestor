package s3upload

import (
	"context"
	"fmt"
	"os"
	"path"

	"github.com/SwissOpenEM/Ingestor/internal/transfertask"
	"github.com/google/uuid"
	openapi_types "github.com/oapi-codegen/runtime/types"

	"golang.org/x/oauth2"
	"golang.org/x/sync/errgroup"
)

type S3Objects struct {
	Files       []string
	ObjectNames []string
	TotalBytes  int64
}

func GetTokens(ctx context.Context, endpoint string, userToken string) (string, string, error) {
	resp, err := GetPresignedUrlServer(endpoint).CreateNewServiceTokenWithResponse(ctx,
		createAddAuthorizationHeaderFunction(userToken))

	if err != nil {
		return "", "", err
	}

	if resp.HTTPResponse.StatusCode != 201 {
		return "", "", fmt.Errorf("failed to get access tokens: %d, %s", resp.HTTPResponse.StatusCode, resp.HTTPResponse.Status)
	}

	return resp.JSON201.AccessToken, resp.JSON201.RefreshToken, nil
}

func createTokenSource(ctx context.Context, clientID string, tokenUrl string, accessToken string, refreshToken string) oauth2.TokenSource {
	config := &oauth2.Config{
		ClientID: clientID,
		Endpoint: oauth2.Endpoint{TokenURL: tokenUrl},
	}

	token := &oauth2.Token{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
	}

	return config.TokenSource(ctx, token)
}

// Upload all files in a folder using presinged urls
func UploadS3(ctx context.Context, task *transfertask.TransferTask, options transfertask.S3TransferConfig, accessToken string, refreshToken string, notifier transfertask.ProgressNotifier) error {

	if len(task.GetFileList()) == 0 {
		return fmt.Errorf("empty file list provided")
	}

	datasetFolder := task.DatasetFolder.FolderPath
	datasetId := task.GetDatasetId()
	uploadId := task.DatasetFolder.Id

	s3Objects := S3Objects{}
	for _, f := range task.GetFileList() {
		info, _ := os.Stat(path.Join(datasetFolder, f.Path))
		if info.IsDir() {
			continue
		}
		s3Objects.TotalBytes += info.Size()
		s3Objects.Files = append(s3Objects.Files, path.Join(datasetFolder, f.Path))
		s3Objects.ObjectNames = append(s3Objects.ObjectNames, "openem-network/datasets/"+datasetId+"/raw_files/"+f.Path)
	}

	transferNotifier := transfertask.NewTransferNotifier(s3Objects.TotalBytes, uploadId, notifier, task)

	task.TransferStarted()
	tokenSource := createTokenSource(context.Background(), options.ClientID, options.TokenUrl, accessToken, refreshToken)

	return uploadFiles(ctx, &s3Objects, options, &transferNotifier, uploadId, tokenSource)
}

func uploadFiles(ctx context.Context, s3Objects *S3Objects, options transfertask.S3TransferConfig, transferNotifier *transfertask.TransferNotifier, uploadId uuid.UUID, tokenSource oauth2.TokenSource) error {
	errorGroup, context := errgroup.WithContext(ctx)
	objectsChannel := make(chan int, len(s3Objects.Files))

	nWorkers := max(options.ConcurrentFiles, len(s3Objects.Files))

	for t := 0; t < nWorkers; t++ {
		errorGroup.Go(
			func() error {
				for idx := range objectsChannel {
					select {
					case <-context.Done():
						transferNotifier.OnTaskCanceled(uploadId)
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

func FinalizeUpload(ctx context.Context, config transfertask.S3TransferConfig, dataset_pid string, ownerUser string, ownerGroup string, email string, autoArchive bool, accessToken string, refreshToken string) error {

	tokenSource := createTokenSource(ctx, config.ClientID, config.TokenUrl, accessToken, refreshToken)

	token, err := tokenSource.Token()
	if err != nil {
		return fmt.Errorf("error fetching a new token: %w", err)
	}

	resp, err := GetPresignedUrlServer(config.Endpoint).FinalizeDatasetUploadWithResponse(ctx, FinalizeDatasetUploadBody{
		DatasetPID:         dataset_pid,
		OwnerUser:          ownerUser,
		OwnerGroup:         ownerGroup,
		ContactEmail:       openapi_types.Email(email),
		CreateArchivingJob: autoArchive,
	}, createAddAuthorizationHeaderFunction(token.AccessToken))

	if err != nil {
		return err
	}

	if resp.HTTPResponse.StatusCode == 500 {
		return fmt.Errorf("failed to finalize upload: %d, %s, %s ", resp.HTTPResponse.StatusCode, resp.HTTPResponse.Status, *resp.JSON500.Details)
	} else if resp.HTTPResponse.StatusCode == 201 {
		logger.Debug("Upload finalized", "dataset pid", resp.JSON201.DatasetID, "message", resp.JSON201.Message)
	} else {
		return fmt.Errorf("failed to finalize upload: %d, %s", resp.HTTPResponse.StatusCode, resp.HTTPResponse.Status)
	}

	return nil
}
