package transfertask

type S3TransferConfig struct {
	ClientID        string `string:"ClientID"`
	TokenURL        string `string:"TokenUrl" validate:"http_url"`
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
	CollectionRootPath      string   `yaml:"collectionRootPath,omitempty"`
	DestinationCollectionID string   `yaml:"destinationCollection"`
	DestinationTemplate     string   `yaml:"destinationTemplate"`
}

type ExtGlobusTransferConfig struct {
	TransferServiceURL string `string:"TransferServiceUrl" validate:"http_url"`
	CollectionRootPath string `string:"CollectionRootPath"`
	SrcFacility        string `string:"SourceFacility" validate:"required"`
	DstFacility        string `string:"DestinationFacility" validate:"required"`
}

type TransferConfig struct {
	Method           string                  `string:"Method" validate:"oneof=S3 Globus ExtGlobus None"`
	StorageLocation  string                  `string:"StorageLocation"`
	ConcurrencyLimit int                     `int:"ConcurrencyLimit" validate:"gte=0"`
	QueueSize        int                     `int:"QueueSize"`
	S3               S3TransferConfig        `mapstructure:"S3" validate:"required_if=Method S3,omitempty"`
	Globus           GlobusTransferConfig    `mapstructure:"Globus" validate:"required_if=Method Globus,omitempty"`
	ExtGlobus        ExtGlobusTransferConfig `mapstrcuture:"ExtGlobus" validate:"required_if=Method ExtGlobus,omitempty"`
}
