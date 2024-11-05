package metadataextractor

import (
	"html/template"
	"os"
	"path"
	"reflect"
	"testing"

	"github.com/labstack/gommon/log"
)

func TestNewExtractorHandler(t *testing.T) {
	type args struct {
		config ExtractorsConfig
	}
	tests := []struct {
		name string
		args args
		want *ExtractorHandler
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := NewExtractorHandler(tt.args.config); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NewExtractorHandler() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestMetadataFilePath(t *testing.T) {
	type args struct {
		folder string
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := MetadataFilePath(tt.args.folder); got != tt.want {
				t.Errorf("MetadataFilePath() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_downloadRelease(t *testing.T) {
	type args struct {
		github_org   string
		github_proj  string
		version      string
		targetFolder string
	}
	tests := []struct {
		name    string
		args    args
		want    string
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := downloadRelease(tt.args.github_org, tt.args.github_proj, tt.args.version, tt.args.targetFolder)
			if (err != nil) != tt.wantErr {
				t.Errorf("downloadRelease() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("downloadRelease() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_verifyFile(t *testing.T) {
	type args struct {
		file_path string
		config    ExtractorConfig
	}
	tests := []struct {
		name    string
		args    args
		want    bool
		want1   string
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, got1, err := verifyFile(tt.args.file_path, tt.args.config)
			if (err != nil) != tt.wantErr {
				t.Errorf("verifyFile() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("verifyFile() got = %v, want %v", got, tt.want)
			}
			if got1 != tt.want1 {
				t.Errorf("verifyFile() got1 = %v, want %v", got1, tt.want1)
			}
		})
	}
}

func Test_downloadExtractor(t *testing.T) {
	t.SkipNow()
	type args struct {
		full_install_path string
		config            ExtractorConfig
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := downloadExtractor(tt.args.full_install_path, tt.args.config); (err != nil) != tt.wantErr {
				t.Errorf("downloadExtractor() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestExtractorHandler_Extractors(t *testing.T) {
	type fields struct {
		extractors   map[string]Extractor
		outputFolder string
	}
	tests := []struct {
		name   string
		fields fields
		want   [](string)
	}{
		{
			name: "TestExtractorHandler_Extractors",
			fields: fields{
				extractors: map[string]Extractor{
					"ext1": {},
					"ext2": {},
				},
				outputFolder: "/some/path",
			},
			want: []string{
				"ext1",
				"ext2",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := &ExtractorHandler{
				extractors:   tt.fields.extractors,
				outputFolder: tt.fields.outputFolder,
			}
			if got := e.Extractors(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ExtractorHandler.Extractors() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_buildCommandline(t *testing.T) {

	templ, _ := template.New("name").Parse("{{.SourceFolder}} {{.OutputFile}} {{.AdditionalParameters}}")

	type args struct {
		templ           *template.Template
		template_params ExtractorInvokationParameters
	}
	tests := []struct {
		name    string
		args    args
		want    string
		want1   []string
		wantErr bool
	}{
		{
			name: "AdditionalParameters",
			args: args{
				templ: templ,
				template_params: ExtractorInvokationParameters{
					Executable:           "test.exe",
					SourceFolder:         "/path/to/sourcefolder",
					OutputFile:           "/path/to/output.json",
					AdditionalParameters: "-f someParam -g someOtherParam",
				},
			},
			want: "test.exe",
			want1: []string{
				"/path/to/sourcefolder",
				"/path/to/output.json",
				"-f",
				"someParam",
				"-g",
				"someOtherParam",
			},
		},
		{
			name: "NoAdditionalParameters",
			args: args{
				templ: templ,
				template_params: ExtractorInvokationParameters{
					Executable:   "test.exe",
					SourceFolder: "/path/to/sourcefolder",
					OutputFile:   "/path/to/output.json",
				},
			},
			want: "test.exe",
			want1: []string{
				"/path/to/sourcefolder",
				"/path/to/output.json",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, got1, err := buildCommandline(tt.args.templ, tt.args.template_params)
			if (err != nil) != tt.wantErr {
				t.Errorf("buildCommandline() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("buildCommandline() got = %v, want %v", got, tt.want)
			}
			if !reflect.DeepEqual(got1, tt.want1) {
				t.Errorf("buildCommandline() got1 = %v, want %v", got1, tt.want1)
			}
		})
	}
}

func stdout_callback(m string) { log.Info(m) }
func stderr_callback(m string) { log.Error(m) }

func Test_runExtractor(t *testing.T) {
	type args struct {
		executable string
		args       []string
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "EchoTest",
			args: args{
				executable: "echo",
				args: []string{
					"hello",
					"world",
				},
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := runExtractor(tt.args.executable, tt.args.args, stdout_callback, stderr_callback); (err != nil) != tt.wantErr {
				t.Errorf("runExtractor() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestExtractorHandler_ExtractMetadata(t *testing.T) {
	// TODO: make platform independent
	templ, _ := template.New("name").Parse("{{.OutputFile}}")

	type fields struct {
		extractors   map[string]Extractor
		outputFolder string
	}
	type args struct {
		extractor_name string
		folder         string
		output_file    string
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    string
		wantErr bool
	}{
		{
			name: "touchExtractor",
			fields: fields{
				extractors: map[string]Extractor{
					"touchExtractor": {
						ExecutablePath: "touch",
						templ:          templ,
					},
				},
			},
			args: args{
				extractor_name: "touchExtractor",
				folder:         "./",
				output_file:    path.Join(os.TempDir(), "output.txt"),
			},
			want:    "", // size of a directory on linux
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := &ExtractorHandler{
				extractors:   tt.fields.extractors,
				outputFolder: tt.fields.outputFolder,
			}
			got, err := e.ExtractMetadata(tt.args.extractor_name, tt.args.folder, tt.args.output_file, stdout_callback, stderr_callback)
			if (err != nil) != tt.wantErr {
				t.Errorf("ExtractorHandler.ExtractMetadata() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("ExtractorHandler.ExtractMetadata() = %v, want %v", got, tt.want)
			}
		})
	}
}
