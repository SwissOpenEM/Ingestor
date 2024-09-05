package core

import (
	"context"
	"log"
	"os"
	"path"
	"time"

	"github.com/google/uuid"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

// Progress notifier object for Minio upload
type MinioProgressNotifier struct {
	total_file_size     int64
	current_size        int64
	files_count         int
	current_file        int
	previous_percentage float64
	start_time          time.Time
	id                  uuid.UUID
	notifier            ProgressNotifier
}

// Callback that gets called by fputobject.
// Note: does not work for multipart uploads
func (pn *MinioProgressNotifier) Read(p []byte) (n int, err error) {
	n = len(p)
	pn.current_size += int64(n)

	pn.notifier.OnTaskProgress(pn.id, pn.current_file, pn.files_count, int(time.Since(pn.start_time).Seconds()))
	return
}

// Upload all files in a folder to a minio bucket
func UploadS3(task_ctx context.Context, dataset_pid string, datasetSourceFolder string, uploadId uuid.UUID, options S3, notifier ProgressNotifier) (string, error) {
	accessKeyID := options.User
	secretAccessKey := options.Password
	creds := credentials.NewStaticV4(accessKeyID, secretAccessKey, "")
	useSSL := false

	log.Printf("Using endpoint %s\n", options.Endpoint)

	// Initialize minio client object.
	minioClient, err := minio.New(options.Endpoint, &minio.Options{
		Creds:  creds,
		Secure: useSSL,
	})

	if err != nil {
		log.Fatalln(err)
	}

	// Make a new bucket called testbucket.
	bucketName := options.Bucket

	err = minioClient.MakeBucket(task_ctx, bucketName, minio.MakeBucketOptions{Region: options.Location})
	if err != nil {
		// Check to see if we already own this bucket (which happens if you run this twice)
		exists, errBucketExists := minioClient.BucketExists(task_ctx, bucketName)
		if errBucketExists == nil && exists {
			log.Printf("We already own %s\n", bucketName)
		} else {
			log.Fatalln(err)
		}
	} else {
		log.Printf("Successfully created %s\n", bucketName)
	}

	contentType := "application/octet-stream"

	entries, err := os.ReadDir(datasetSourceFolder)
	if err != nil {
		return "", err
	}

	pn := MinioProgressNotifier{files_count: len(entries), previous_percentage: 0.0, start_time: time.Now(), id: uploadId, notifier: notifier}

	for idx, f := range entries {
		select {
		case <-task_ctx.Done():
			pn.notifier.OnTaskCanceled(uploadId)
			return "Upload canceled", nil

		default:
			filePath := path.Join(datasetSourceFolder, f.Name())
			objectName := "openem-network/datasets/" + dataset_pid + "/raw_files/" + f.Name()

			pn.current_file = idx + 1
			fileinfo, _ := os.Stat(filePath)
			pn.total_file_size = fileinfo.Size()

			notifier.OnTaskProgress(uploadId, pn.current_file, pn.files_count, 0)

			_, err := minioClient.FPutObject(task_ctx, bucketName, objectName, filePath, minio.PutObjectOptions{
				ContentType:           contentType,
				Progress:              &pn,
				SendContentMd5:        true,
				NumThreads:            4,
				DisableMultipart:      false,
				ConcurrentStreamParts: true,
			})
			if err != nil {
				return dataset_pid, err
			}
		}
	}

	return dataset_pid, nil
}
