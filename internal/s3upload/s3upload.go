package s3upload

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path"
	"sync/atomic"
	"time"

	"github.com/SwissOpenEM/Ingestor/internal/task"
	"github.com/google/uuid"

	"github.com/paulscherrerinstitute/scicat-cli/v3/datasetIngestor"
	"golang.org/x/oauth2"
	"golang.org/x/sync/errgroup"
)

// Progress notifier object for Minio upload
type TransferNotifier struct {
	totalBytes     int64
	bytesTansfered int64
	FilesCount     int
	startTime      time.Time
	id             uuid.UUID
	notifier       task.ProgressNotifier
	TaskStatus     *task.Status
}

func (pn *TransferNotifier) AddUploadedBytes(numBytes int64) {
	atomic.AddInt64(&pn.bytesTansfered, numBytes)
}

func (pn *TransferNotifier) UpdateTaskProgress() {
	t := time.Since(pn.startTime)
	pn.notifier.OnTaskProgress(pn.id, float32(pn.bytesTansfered)/float32(pn.totalBytes)*100, int(t.Seconds()))
}

type S3Objects struct {
	Files       []string
	ObjectNames []string
	TotalBytes  int64
}

func getTokens(ctx context.Context, endpoint string, userToken string) (string, string, error) {
	resp, err := GetPresignedUrlServer(endpoint).CreateNewServiceTokenWithResponse(ctx,
		createAddAuthorizationHeaderFunction(userToken))

	if err != nil {
		return "", "", err
	}

	if resp.HTTPResponse.StatusCode != 201 {
		slog.Error(fmt.Sprintf("Error getting new access token: %d, %s", resp.HTTPResponse.StatusCode, resp.HTTPResponse.Status))
		return "", "", fmt.Errorf("failed to get access tokens: %s", resp.HTTPResponse.Status)
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
func UploadS3(ctx context.Context, datasetPID string, datasetSourceFolder string, fileList []datasetIngestor.Datafile, uploadId uuid.UUID, options task.S3TransferConfig, userToken string, notifier task.ProgressNotifier) error {

	if len(fileList) == 0 {
		return fmt.Errorf("empty file list provided")
	}

	s3Objects := S3Objects{}
	for _, f := range fileList {
		s, _ := os.Stat(path.Join(datasetSourceFolder, f.Path))
		s3Objects.TotalBytes += s.Size()
		s3Objects.Files = append(s3Objects.Files, path.Join(datasetSourceFolder, f.Path))
		s3Objects.ObjectNames = append(s3Objects.ObjectNames, "openem-network/datasets/"+datasetPID+"/raw_files/"+f.Path)
	}

	transferNotifier := TransferNotifier{totalBytes: s3Objects.TotalBytes, bytesTansfered: 0, startTime: time.Now(), id: uploadId, notifier: notifier}

	accessToken, refreshToken, err := getTokens(ctx, options.Endpoint, userToken)
	if err != nil {
		slog.Error(fmt.Sprintf("Failed to fetch token from '%s': %v", options.Endpoint, err))
		return err
	}

	tokenSource := createTokenSource(context.Background(), options.ClientID, options.TokenUrl, accessToken, refreshToken)

	err = uploadFiles(ctx, &s3Objects, options, &transferNotifier, uploadId, tokenSource)

	return err
}

func uploadFiles(ctx context.Context, s3Objects *S3Objects, options task.S3TransferConfig, transferNotifier *TransferNotifier, uploadId uuid.UUID, tokenSource oauth2.TokenSource) error {
	errorGroup, context := errgroup.WithContext(ctx)
	objectsChannel := make(chan int, len(s3Objects.Files))

	nWorkers := max(options.ConcurrentFiles, len(s3Objects.Files))

	for t := 0; t < nWorkers; t++ {
		errorGroup.Go(
			func() error {
				for idx := range objectsChannel {
					select {
					case <-context.Done():
						transferNotifier.notifier.OnTaskCanceled(uploadId)
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
