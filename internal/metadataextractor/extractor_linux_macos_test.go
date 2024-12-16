//go:build !windows
// +build !windows

package metadataextractor

import (
	"context"
	"html/template"
	"os"
	"path"
	"runtime"
	"testing"
	"time"
)

func TestExtractorHandler_ExtractMetadata(t *testing.T) {
	// TODO: make platform independent
	templ, _ := template.New("name").Parse("{} {{.OutputFile}}")
	_, goFile, _, ok := runtime.Caller(0)
	if !ok {
		return // skip test if can't get path
	}
	currDir := path.Dir(goFile)
	execPath := path.Join(currDir, "extractor_test_echoToFile.sh")

	type fields struct {
		methods      map[string]Method
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
			name: "echoExtractor",
			fields: fields{
				methods: map[string]Method{"echoExtractor": {Name: "echoExtractor", Schema: "someschema", Extractor: "echoExtractor"}},
				extractors: map[string]Extractor{
					"echoExtractor": {
						ExecutablePath: execPath,
						templ:          templ,
					},
				},
			},
			args: args{
				extractor_name: "echoExtractor",
				folder:         "./",
				output_file:    path.Join(os.TempDir(), "output.txt"),
			},
			want:    "{}\n", // size of a directory on linux
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := &ExtractorHandler{
				methods:      tt.fields.methods,
				extractors:   tt.fields.extractors,
				outputFolder: tt.fields.outputFolder,
				timeout:      time.Minute,
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
