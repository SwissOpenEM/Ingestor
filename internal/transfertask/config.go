package transfertask

type S3TransferConfig struct {
	ClientID        string `string:"ClientID"`
	TokenUrl        string `string:"TokenUrl" validate:"http_url"`
	Endpoint        string `string:"Endpoint" validate:"http_url"`
	ChunkSizeMB     int64  `int64:"ChunkSizeMB" validate:"required"`
	ConcurrentFiles int    `int:"ConcurrentFiles" validate:"required"`
	PoolSize        int    `int:"PoolSize" validate:"required"`
}

type GlobusTransferConfig struct {
	ClientID                string   `yaml:"clientId"`
	ClientSecret            string   `yaml:"clientSecret,omitempty"`
	RedirectURL             string   `yaml:"redirectUrl"`
	Scopes                  []string `yaml:"scopes,omitempty"`
	SourceCollectionID      string   `yaml:"sourceCollection"`
	SourcePrefixPath        string   `yaml:"sourcePrefixPath,omitempty"`
	DestinationCollectionID string   `yaml:"destinationCollection"`
	DestinationTemplate     string   `yaml:"destinationTemplate"`
}

type TransferConfig struct {
	Method           string               `string:"Method" validate:"oneof=S3 Globus"`
	ConcurrencyLimit int                  `int:"ConcurrencyLimit" validate:"gte=0"`
	QueueSize        int                  `int:"QueueSize"`
	S3               S3TransferConfig     `mapstructure:"S3" validate:"required_if=Method S3,omitempty"`
	Globus           GlobusTransferConfig `mapstructure:"Globus" validate:"required_if=Method Globus,omitempty"`
}
