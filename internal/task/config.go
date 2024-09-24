package task

type S3TransferConfig struct {
	Endpoint string `string:"Endpoint"`
	Bucket   string `string:"Bucket"`
	Location string `string:"Location"`
	User     string `string:"User"`
	Password string `string:"Password"`
	Checksum bool   `bool:"Checksum"`
}

type GlobusTransferConfig struct {
	ClientID              string   `yaml:"clientId"`
	ClientSecret          string   `yaml:"clientSecret,omitempty"`
	RedirectURL           string   `yaml:"redirectUrl"`
	Scopes                []string `yaml:"scopes,omitempty"`
	SourceCollection      string   `yaml:"sourceCollection"`
	SourcePrefixPath      string   `yaml:"sourcePrefixPath,omitempty"`
	DestinationCollection string   `yaml:"destinationCollection"`
	DestinationPrefixPath string   `yaml:"destinationPrefixPath,omitempty"`
	RefreshToken          string   `yaml:"refreshToken,omitempty"`
}

type TransferConfig struct {
	Method string               `string:"method"`
	S3     S3TransferConfig     `mapstructure:"s3"`
	Globus GlobusTransferConfig `mapstructure:"globus"`
}
