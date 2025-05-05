package core

import (
	"reflect"
	"testing"
	"time"

	"github.com/SwissOpenEM/Ingestor/internal/metadataextractor"
	"github.com/SwissOpenEM/Ingestor/internal/transfertask"
	"github.com/SwissOpenEM/Ingestor/internal/webserver/wsconfig"
	"github.com/spf13/viper"
)

func createExpectedValidConfigS3() transfertask.TransferConfig {
	return transfertask.TransferConfig{
		Method:           "S3",
		StorageLocation:  "SomeFacility",
		ConcurrencyLimit: 10,
		QueueSize:        1000,
		S3: transfertask.S3TransferConfig{
			Endpoint:        "https://endpoint/api/v1",
			TokenUrl:        "https://keycloak.localhost/realms/facility/protocol/openid-connect/token",
			ClientID:        "archiver-service-api",
			ChunkSizeMB:     64,
			ConcurrentFiles: 4,
			PoolSize:        8,
		},
	}
}

func createExpectedValidConfigGlobus() transfertask.TransferConfig {
	return transfertask.TransferConfig{
		Method:           "Globus",
		StorageLocation:  "SomeFacility",
		ConcurrencyLimit: 10,
		QueueSize:        1000,
		Globus: transfertask.GlobusTransferConfig{
			ClientID:                "clientid_registered_with_globus",
			RedirectURL:             "https://auth.globus.org/v2/web/auth-code",
			Scopes:                  []string{"urn:globus:auth:scope:transfer.api.globus.org:all[*https://auth.globus.org/scopes/[collection_id1]/data_access]"},
			SourceCollectionID:      "collectionid1",
			SourcePrefixPath:        "/some/optional/path",
			DestinationCollectionID: "collectionid2",
			DestinationTemplate:     "/{{ .Username }}/{{ replace .Pid \".\" \"_\" }}/{{ .DatasetFolder }}",
		},
	}
}

func createExpectedValidConfig(transferConfig transfertask.TransferConfig) Config {
	expected_scicat := ScicatConfig{
		Host: "http://scicat:8080/api/v3",
	}

	expected_tranfer := transferConfig

	expected_ws := wsconfig.WebServerConfig{
		AuthConf: wsconfig.AuthConf{
			SessionDuration: 28800,
			FrontendConf: wsconfig.FrontendConf{
				Origin:       "http://scicat.localhost",
				RedirectPath: "/ingestor",
			},
			OAuth2Conf: wsconfig.OAuth2Conf{
				ClientID:    "ingestor",
				RedirectURL: "http://localhost:8888/callback",
				Scopes:      []string{"email"},
			},
			OIDCConf: wsconfig.OIDCConf{
				IssuerURL: "http://keycloak.localhost/realms/facility",
			},
			JWTConf: wsconfig.JWTConf{
				UseJWKS:              true,
				JwksURL:              "http://keycloak.localhost/realms/facility/protocol/openid-connect/certs",
				JwksSignatureMethods: []string{"RS256"},
			},
			RBACConf: wsconfig.RBACConf{
				AdminRole:             "FACILITY-ingestor-admin",
				CreateModifyTasksRole: "FACILITY-ingestor-write",
				ViewTasksRole:         "FACILITY-ingestor-read",
			},
		},
		PathsConf: wsconfig.PathsConf{
			CollectionLocation: "/some/path",
		},
		MetadataExtJobsConf: wsconfig.MetadataExtJobsConf{
			ConcurrencyLimit: 100,
			QueueSize:        200,
		},
		OtherConf: wsconfig.OtherConf{
			Port:     8888,
			LogLevel: "Info",
		},
	}

	expected_LS_methods := []metadataextractor.MethodConfig{
		{
			Name:   "Single Particle",
			Schema: "singleParticleSchema.json",
		},
		{
			Name:   "Cellular Tomography",
			Schema: "cellularTomographySchema.json",
		},
		{
			Name:   "Tomography",
			Schema: "tomographySchema.json",
		},
		{
			Name:   "Environmental Tomography",
			Schema: "environmentalTomographySchema.json",
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
				CommandLineTemplate:  "-i '{{.SourceFolder}}' -o '{{.OutputFile}}' {{.AdditionalParameters}}",
				AdditionalParameters: []string{"--param1=value1", "--param2=value2"},
				Methods:              expected_LS_methods,
			},
			{
				Name:                "MS",
				GithubOrg:           "SwissOpenEM",
				GithubProject:       "MS_Metadata_reader",
				Version:             "v0.9.9",
				Executable:          "MS_Metadata_reader",
				Checksum:            "d7052dec32d99f35bcbe95d780afb949585c33b5e538a4754611f7f1ead1c0ba",
				ChecksumAlg:         "sha256",
				CommandLineTemplate: "-i '{{.SourceFolder}}' -o '{{.OutputFile}}' {{.AdditionalParameters}}",
				Methods: []metadataextractor.MethodConfig{
					{
						Name:   "Material Science",
						Schema: "some.json",
					},
				},
			},
		},
		InstallationPath:          "./parentPathToAllExtractors/",
		DownloadMissingExtractors: false,
		SchemasLocation:           "./ExtractorSchemas",
		Timeout:                   time.Minute * 4,
	}

	expected_config := Config{
		MetadataExtractors: expected_meta,
		Scicat:             expected_scicat,
		Transfer:           expected_tranfer,
		WebServer:          expected_ws,
	}
	return expected_config
}

func TestReadConfigS3(t *testing.T) {
	viperTestConf := viper.New()
	viperTestConf.SetConfigType("yaml")
	viperTestConf.AddConfigPath("../../test/testdata")
	configReader := ConfigReader{viperConf: viperTestConf}
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
				configFileName: "valid_config_s3.yaml",
			},
			want:    createExpectedValidConfig(createExpectedValidConfigS3()),
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := configReader.ReadConfig(tt.args.configFileName)
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

func TestReadConfigGlobus(t *testing.T) {
	viperTestConf := viper.New()
	viperTestConf.SetConfigType("yaml")
	viperTestConf.AddConfigPath("../../test/testdata")
	configReader := ConfigReader{viperConf: viperTestConf}

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
				configFileName: "valid_config_globus.yaml",
			},
			want:    createExpectedValidConfig(createExpectedValidConfigGlobus()),
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := configReader.ReadConfig(tt.args.configFileName)
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

func TestExampleConfig(t *testing.T) {
	viperTestConf := viper.New()
	viperTestConf.SetConfigType("yaml")
	viperTestConf.AddConfigPath("../../configs")
	configReader := ConfigReader{viperConf: viperTestConf}

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
				configFileName: "openem-ingestor-config.yaml",
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := configReader.ReadConfig(tt.args.configFileName)
			if (err != nil) != tt.wantErr {
				t.Errorf("ReadConfig() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
		})
	}
}
