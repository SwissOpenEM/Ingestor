package core

import (
	"reflect"
	"testing"

	"github.com/SwissOpenEM/Ingestor/internal/metadataextractor"
	"github.com/SwissOpenEM/Ingestor/internal/task"
)

func createExpectedValidConfig() Config {
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

	expected_meta := metadataextractor.ExtractorsConfig{
		Extractors: []metadataextractor.ExtractorConfig{
			{
				Name:                 "LS",
				GithubOrg:            "SwissOpenEM",
				GithubProject:        "LS_Metadata_reader",
				Version:              "v0.2.3",
				Executable:           "LS_Metadata_reader",
				Checksum:             "8c5249c41a5b3464d183d063be7d96d9557dcb11c76598690f2c20bb06937fbe",
				ChecksumAlg:          "sha256",
				CommandLineTemplate:  "-i {{.SourceFolder}} -o {{.OutputFile}} {{.AdditionalParameters}}",
				AdditionalParameters: []string{"--param1=value1", "--param2=value2"},
			},
			{
				Name:                "MS",
				GithubOrg:           "SwissOpenEM",
				GithubProject:       "MS_Metadata_reader",
				Version:             "v0.9.9",
				Executable:          "MS_Metadata_reader",
				Checksum:            "d7052dec32d99f35bcbe95d780afb949585c33b5e538a4754611f7f1ead1c0ba",
				ChecksumAlg:         "sha256",
				CommandLineTemplate: "-i {{.SourceFolder}} -o {{.OutputFile}} {{.AdditionalParameters}}",
			},
		},
		InstallationPath:          "./parentPathToAllExtractors/",
		DownloadMissingExtractors: false,
	}

	expected_config := Config{
		Misc:               expected_misc,
		MetadataExtractors: expected_meta,
		Scicat:             expected_scicat,
		Transfer:           expected_s3,
	}
	return expected_config
}

func TestReadConfig(t *testing.T) {
	viperConf.AddConfigPath("../../test/testdata")

	type args struct {
		configFileName string
	}
	tests := []struct {
		name    string
		args    args
		want    Config
		wantErr bool
	}{
		{
			name: "valid config file",
			args: args{
				configFileName: "valid_config.yaml",
			},
			want:    createExpectedValidConfig(),
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ReadConfig(tt.args.configFileName)
			if (err != nil) != tt.wantErr {
				t.Errorf("ReadConfig() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ReadConfig() = %v, want %v", got, tt.want)
			}
		})
	}
}
