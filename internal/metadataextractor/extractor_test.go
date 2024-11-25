package metadataextractor

import (
	"context"
	b64 "encoding/base64"
	"html/template"
	"log"
	"os"
	"path"
	"reflect"
	"testing"
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

func TestExtractorHandler_Methods(t *testing.T) {

	// create mock schemas
	schemaPath := path.Join(t.TempDir(), "testSchema.json")
	testSchema := "{\"name\":\"testSchema\"}"
	//nolint:all
	os.WriteFile(schemaPath, []byte(testSchema), 0644)

	// create mock extractors
	extractorsPath := t.TempDir()

	ex1 := path.Join(extractorsPath, "ex1")
	//nolint:all
	os.WriteFile(ex1, []byte(""), 0644)

	ex2 := path.Join(extractorsPath, "ex2")
	//nolint:all
	os.WriteFile(ex2, []byte(""), 0644)

	type fields struct {
		config ExtractorsConfig
	}
	tests := []struct {
		name   string
		fields fields
		want   [](MethodAndSchema)
	}{
		{
			name: "TestExtractorHandler_Methods",
			fields: fields{
				config: ExtractorsConfig{
					DownloadMissingExtractors: false,
					SchemasLocation:           path.Dir(schemaPath),
					InstallationPath:          extractorsPath,
					Extractors: []ExtractorConfig{
						{
							Name:                "ex1",
							Executable:          "ex1",
							CommandLineTemplate: "",
							Methods: []MethodConfig{
								{Name: "ex1_method1", Schema: path.Base(schemaPath)},
								{Name: "ex1_method2", Schema: path.Base(schemaPath)},
							},
						},
						{
							Name:                "ex2",
							Executable:          "ex2",
							CommandLineTemplate: "",
							Methods: []MethodConfig{
								{Name: "ex2_method1", Schema: path.Base(schemaPath)},
								{Name: "ex2_method2", Schema: path.Base(schemaPath)},
							},
						},
					},
				},
			},
			want: []MethodAndSchema{
				{
					Name:   "ex1_method1",
					Schema: b64.StdEncoding.EncodeToString([]byte(testSchema)),
				},
				{
					Name:   "ex1_method2",
					Schema: b64.StdEncoding.EncodeToString([]byte(testSchema)),
				},
				{
					Name:   "ex2_method1",
					Schema: b64.StdEncoding.EncodeToString([]byte(testSchema)),
				},
				{
					Name:   "ex2_method2",
					Schema: b64.StdEncoding.EncodeToString([]byte(testSchema)),
				},
			},
		},
		{
			name: "TestExtractorHandler_Methods_Collision",
			fields: fields{
				config: ExtractorsConfig{
					DownloadMissingExtractors: false,
					SchemasLocation:           path.Dir(schemaPath),
					InstallationPath:          extractorsPath,
					Extractors: []ExtractorConfig{
						{
							Name:                "ex1",
							Executable:          "ex1",
							CommandLineTemplate: "",
							Methods: []MethodConfig{
								{Name: "ex1_method1", Schema: path.Base(schemaPath)},
							},
						},
						{
							Name:                "ex2",
							Executable:          "ex2",
							CommandLineTemplate: "",
							Methods: []MethodConfig{
								{Name: "ex1_method1", Schema: path.Base(schemaPath)},
							},
						},
					},
				},
			},
			want: []MethodAndSchema{
				{
					Name:   "ex1_method1",
					Schema: b64.StdEncoding.EncodeToString([]byte(testSchema)),
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := NewExtractorHandler(
				tt.fields.config,
			)
			if got := e.AvailableMethods(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ExtractorHandler.AvailableMethods() = %v, want %v", got, tt.want)
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

func stdout_callback(m string) { log.Print(m) }
func stderr_callback(m string) { log.Print(m) }

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
			ctx := context.Background()
			if err := runExtractor(ctx, tt.args.executable, tt.args.args, stdout_callback, stderr_callback); (err != nil) != tt.wantErr {
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
			ctx := context.Background()
			got, err := e.ExtractMetadata(ctx, tt.args.extractor_name, tt.args.folder, tt.args.output_file, stdout_callback, stderr_callback)
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

func TestSplitString(t *testing.T) {
	type args struct {
		str string
		r   rune
	}
	tests := []struct {
		name string
		args args
		want []string
	}{
		{
			name: "No whitespace",
			args: args{
				str: "-i 'filename_without_whitespace' -o 'filename'",
				r:   ' ',
			},
			want: []string{
				"-i", "'filename_without_whitespace'", "-o", "'filename'",
			},
		},
		{
			name: "With whitespace",
			args: args{
				str: "-i 'filename with whitespaces' -o 'filename'",
				r:   ' ',
			},
			want: []string{
				"-i", "'filename with whitespaces'", "-o", "'filename'",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := SplitString(tt.args.str, tt.args.r); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("SplitString() = %v, want %v", got, tt.want)
			}
		})
	}
}
