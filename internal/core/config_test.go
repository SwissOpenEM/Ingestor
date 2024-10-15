package core

import (
	"testing"

	"github.com/SwissOpenEM/Ingestor/internal/task"
	"github.com/stretchr/testify/assert"
)

func TestConfig(t *testing.T) {
	viperConf.AddConfigPath("../../test/testdata")

	testConfigFile := "valid_config.yaml"

	err := ReadConfig(testConfigFile)
	assert.NoError(t, err, "File parsing failed")

	config, err := GetConfig()
	assert.NoError(t, err, "File parsing failed")

	expected_misc := MiscConfig{
		ConcurrencyLimit: 2,
		Port:             8888,
	}

	expected_scicat := ScicatConfig{
		Host:        "http://scicat:8080/api/v3",
		AccessToken: "token",
	}

	expected_s3 := task.TransferConfig{
		Method: "S3",
		S3: task.S3TransferConfig{
			Endpoint: "s3:9000",
			Bucket:   "landingzone",
			Location: "eu-west-1",
			User:     "minio_user",
			Password: "minio_pass",
			Checksum: true,
		},
	}


	expected_config := Config{
		Misc:               expected_misc,
		Scicat:             expected_scicat,
		Transfer:           expected_s3,
	}

	assert.Equal(t, expected_config, config)

}
