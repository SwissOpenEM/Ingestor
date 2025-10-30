package s3upload

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"io"
	"math"
	"net/http"
	"os"
	"strings"
	"sync"

	"github.com/SwissOpenEM/Ingestor/internal/transfertask"
	"github.com/alitto/pond/v2"
	"github.com/hashicorp/go-retryablehttp"
	"golang.org/x/oauth2"
)

const (
	MiB = 1024 * 1024
)

type MultipartInput struct {
	File      *os.File
	PartCount int
}

type HTTPUploader struct {
	Pool   pond.Pool
	Client *http.Client
}

var instance *HTTPUploader
var once sync.Once

func InitHTTPUploaderWithPool(pool pond.Pool) {
	once.Do(func() {
		retryClient := retryablehttp.NewClient()
		retryClient.RetryMax = 10
		retryClient.Backoff = retryablehttp.DefaultBackoff
		retryClient.Logger = log()

		standardClient := retryClient.StandardClient()
		instance = &HTTPUploader{Pool: pool, Client: standardClient}
	})
}

func GetHTTPUploader() *HTTPUploader {
	return instance
}

var presignedURLServerClient *ClientWithResponses
var oncePresignedURLServerClient sync.Once

func GetPresignedURLServer(endpoint string) *ClientWithResponses {
	oncePresignedURLServerClient.Do(func() {
		presignedURLServerClient, _ = NewClientWithResponses(endpoint)
	})
	return presignedURLServerClient
}

func createAddAuthorizationHeaderFunction(token string) func(ctx context.Context, req *http.Request) error {
	return func(ctx context.Context, req *http.Request) error {
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))
		return nil
	}
}

// Fetches presigned url(s) from API server. If parts > 1, multipart upload is initiated
func getPresignedURLs(objectName string, part int, endpoint string, userToken string) (string, []string, error) {
	response, err := GetPresignedURLServer(endpoint).GetPresignedUrlsWithResponse(context.Background(), PresignedUrlBody{
		ObjectName: objectName,
		Parts:      part,
	}, createAddAuthorizationHeaderFunction(userToken))

	if err != nil {
		return "", nil, fmt.Errorf("getPresignedURLs: %w", err)
	}

	if response.StatusCode() == http.StatusInternalServerError {
		if response.JSON500 != nil && response.JSON500.Details != nil {
			return "", nil, fmt.Errorf("%s: %s", response.JSON500.Message, *response.JSON500.Details)
		}
		return "", nil, fmt.Errorf("unknown error")
	}
	if response.StatusCode() == http.StatusUnprocessableEntity {
		errString := ""
		if response.JSON422 != nil && response.JSON422.Detail != nil {
			for _, d := range *response.JSON422.Detail {
				errString += " " + d.Msg
			}
		}
		return "", nil, fmt.Errorf("%s", errString)
	}
	return response.JSON201.UploadId, response.JSON201.Urls, nil
}

func completeMultipartUpload(objectName string, uploadID string, endpoint string, parts []CompletePart, fullFileChecksum string, userToken string) error {
	response, err := GetPresignedURLServer(endpoint).CompleteUploadWithResponse(context.Background(), CompleteUploadBody{
		ObjectName:     objectName,
		UploadId:       uploadID,
		Parts:          parts,
		ChecksumSha256: fullFileChecksum,
	}, createAddAuthorizationHeaderFunction(userToken))

	if err != nil {
		return fmt.Errorf("completeMultipartUpload: %w", err)
	}

	if response.StatusCode() == http.StatusInternalServerError {
		if response.JSON500 != nil && response.JSON500.Details != nil {
			return fmt.Errorf("%s: %s", response.JSON500.Message, *response.JSON500.Details)
		}
		return fmt.Errorf("internal server error")
	}
	if response.StatusCode() == http.StatusUnprocessableEntity {
		errString := ""
		if response.JSON422 != nil && response.JSON422.Detail != nil {
			for _, d := range *response.JSON422.Detail {
				errString += " " + d.Msg
			}
		}
		return fmt.Errorf("%s", errString)
	}
	return nil
}

func uploadFile(ctx context.Context, filePath string, objectName string, options transfertask.S3TransferConfig, notifier *transfertask.TransferNotifier, tokenSource oauth2.TokenSource) error {
	file, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("error opening file: %w", err)
	}
	defer file.Close()

	fileInfo, err := file.Stat()
	if err != nil {
		return fmt.Errorf("error getting file info: %w", err)
	}

	totalSize := fileInfo.Size()
	httpClient := GetHTTPUploader()

	token, err := tokenSource.Token()
	if err != nil {
		return fmt.Errorf("error fetching a new token: %w", err)
	}

	if totalSize < options.ChunkSizeMB*MiB {
		return doUploadSingleFile(ctx, objectName, file, httpClient, options.Endpoint, token.AccessToken, notifier)
	}

	uploadID, err := doUploadMultipart(ctx, totalSize, objectName, file, httpClient, options, token.AccessToken, notifier)
	if err != nil {
		errUpload := fmt.Errorf("failed to do multipart upload: uploadID=%s, objectName=%s, error=%s", uploadID, objectName, err.Error())
		errAbort := abortMultipartUpload(uploadID, objectName, options.Endpoint, token.AccessToken)
		if errAbort != nil {
			return fmt.Errorf("while aborting a multipart upload an error occurred: %s. Previous error: %s", errAbort.Error(), errUpload.Error())
		}
		return errUpload
	}
	return nil
}

func abortMultipartUpload(uploadID string, objectName string, endpoint string, userToken string) error {
	response, err := GetPresignedURLServer(endpoint).AbortMultipartUploadWithResponse(context.Background(), AbortUploadBody{
		ObjectName: objectName,
		UploadId:   uploadID,
	}, createAddAuthorizationHeaderFunction(userToken))

	if err != nil {
		return fmt.Errorf("abortMultipartUpload: %w", err)
	}
	if response.StatusCode() != http.StatusOK {
		return fmt.Errorf("abortMultipartUpload: unexpected status code %d", response.StatusCode())
	}
	return nil
}

func doUploadSingleFile(ctx context.Context, objectName string, file *os.File, httpClient *HTTPUploader, endpoint string, userToken string, notifier *transfertask.TransferNotifier) error {
	_, urls, err := getPresignedURLs(objectName, 1, endpoint, userToken)
	if err != nil {
		return fmt.Errorf("doUploadSingleFile: %w", err)
	}

	data, err := io.ReadAll(file)
	if err != nil {
		return fmt.Errorf("doUploadSingleFile: %w", err)
	}
	n := len(data)

	base64Hash, _ := calculateHashB64(&data)
	_, err = uploadData(ctx, &data, urls[0], httpClient, base64Hash)
	if err != nil {
		return fmt.Errorf("doUploadSingleFile: %w", err)
	}
	if ctx.Err() != nil {
		return ctx.Err()
	}
	notifier.AddUploadedBytes(int64(n))
	notifier.IncreaseFileCount(1)
	notifier.UpdateTaskProgress()
	return nil
}

func doUploadMultipart(ctx context.Context, totalSize int64, objectName string, file *os.File, httpClient *HTTPUploader, options transfertask.S3TransferConfig, userToken string, notifier *transfertask.TransferNotifier) (string, error) {
	partCount := int(math.Ceil(float64(totalSize) / float64(options.ChunkSizeMB*MiB)))

	uploadID, presignedURLs, err := getPresignedURLs(objectName, partCount, options.Endpoint, userToken)
	if err != nil {
		return uploadID, fmt.Errorf("doUploadMultipart: %w", err)
	}

	group := httpClient.Pool.NewGroupContext(ctx)
	parts := make([]CompletePart, partCount)
	partChecksums := make([]string, partCount)

	for partNumber := 0; partNumber < partCount; partNumber++ {
		partNumber := partNumber // capture loop variable
		group.SubmitErr(func() error {
			partData := make([]byte, options.ChunkSizeMB*MiB)
			n, err := file.ReadAt(partData, int64(partNumber)*int64(options.ChunkSizeMB*MiB))
			if err != nil && err != io.EOF {
				return fmt.Errorf("doUploadMultipart: %w", err)
			}
			partData = partData[:n]

			base64Hash, hash := calculateHashB64(&partData)
			partChecksums[partNumber] = string(hash[:])
			etag, err := uploadData(ctx, &partData, presignedURLs[partNumber], httpClient, base64Hash)
			if err != nil {
				return fmt.Errorf("doUploadMultipart: %w", err)
			}

			notifier.AddUploadedBytes(int64(n))

			parts[partNumber] = CompletePart{Etag: etag, PartNumber: partNumber + 1, ChecksumSha256: base64Hash}

			return nil
		})
	}

	err = group.Wait()
	if err != nil {
		return uploadID, fmt.Errorf("doUploadMultipart: %w", err)
	}
	if ctx.Err() != nil {
		return uploadID, ctx.Err()
	}

	c := strings.Join(partChecksums, "")
	n := sha256.Sum256([]byte(c))
	base64Hash := base64.StdEncoding.EncodeToString(n[:])

	err = completeMultipartUpload(objectName, uploadID, options.Endpoint, parts, base64Hash, userToken)
	if err != nil {
		return uploadID, fmt.Errorf("error completing multipart upload: %w", err)
	}

	notifier.IncreaseFileCount(1)
	notifier.UpdateTaskProgress()

	return uploadID, nil
}

func calculateHashB64(data *[]byte) (string, [32]byte) {
	hash := sha256.Sum256(*data)
	base64Hash := base64.StdEncoding.EncodeToString(hash[:])
	return base64Hash, hash
}

func uploadData(ctx context.Context, data *[]byte, presignedURL string, httpClient *HTTPUploader, base64Hash string) (string, error) {
	decodedURL, err := base64.StdEncoding.DecodeString(presignedURL)
	if err != nil {
		return "", fmt.Errorf("uploadData: failed to decode presignedURL: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, "PUT", string(decodedURL), bytes.NewReader(*data))
	if err != nil {
		return "", fmt.Errorf("uploadData: %w", err)
	}

	// The checksum algorithm needs to match the one defined in the presigned url
	req.Header.Set("x-amz-checksum-sha256", base64Hash)
	req.Header.Set("Content-Type", "application/json")

	resp, err := httpClient.Client.Do(req)
	if err != nil {
		return "", fmt.Errorf("uploadData: %w", err)
	}
	defer resp.Body.Close()
	etag := strings.ReplaceAll(resp.Header.Get("ETag"), "\"", "")

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("upload failed: %d %s", resp.StatusCode, resp.Status)
	}
	return etag, nil
}
