package metadataextractor

type ExtractorConfig struct {
	Name                 string   `string:"Name"`
	GithubOrg            string   `string:"GithubOrg"`
	GithubProject        string   `string:"GithubProject"`
	Version              string   `string:"Version"`
	Executable           string   `string:"Executable"`
	Checksum             string   `string:"Checksum"`
	ChecksumAlg          string   `string:"ChecksumAlg"`
	CommandLineTemplate  string   `string:"CommandLineTemplate"`
	AdditionalParameters []string `[]string:"AdditionalParameters"`
}

type ExtractorsConfig struct {
	Extractors                []ExtractorConfig `mapstructure:"Extractors" validate:"required"`
	Default                   string            `string:"Default"`
	InstallationPath          string            `string:"InstallationPath" validate:"required"`
	DownloadMissingExtractors bool              `bool:"DownloadMissingExtractors"`
}
