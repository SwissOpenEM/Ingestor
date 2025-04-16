package metadataextractor

import "time"

type MethodConfig struct {
	Name   string `string:"Name" validate:"required"`
	Schema string `string:"Schema" validate:"required"`
	Url    string `string:"Url" validate:"http_url"`
}

type ExtractorConfig struct {
	Name                 string         `string:"Name" validate:"required"`
	GithubOrg            string         `string:"GithubOrg" validate:"required"`
	GithubProject        string         `string:"GithubProject" validate:"required"`
	Version              string         `string:"Version" validate:"required"`
	Executable           string         `string:"Executable" validate:"required"`
	Checksum             string         `string:"Checksum" validate:"required"`
	ChecksumAlg          string         `string:"ChecksumAlg" validate:"required,oneof=sha256"`
	CommandLineTemplate  string         `string:"CommandLineTemplate" validate:"required"`
	AdditionalParameters []string       `[]string:"AdditionalParameters"`
	Methods              []MethodConfig `[]MethodConfig:"Methods" validate:"required,min=1,dive"`
}

type ExtractorsConfig struct {
	Extractors                []ExtractorConfig `[]ExtractorConfig:"Extractors" validate:"dive"` // Enable validation for min=1 again, https://github.com/SwissOpenEM/Ingestor/issues/38
	InstallationPath          string            `string:"InstallationPath" validate:"required"`
	SchemasLocation           string            `string:"SchemasLocation" validate:"required"`
	DownloadMissingExtractors bool              `json:"DownloadMissingExtractors" binding:"required,boolean"`
	DownloadSchemas           bool              `json:"DownloadSchemas" binding:"required,boolean"`
	Timeout                   time.Duration     `string:"Timeout"`
}
