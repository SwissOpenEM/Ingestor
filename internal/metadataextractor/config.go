package metadataextractor

type Extractor struct {
	Name         string `string:"Name"`
	DownloadPath string `string:"DownloadPath"`
	Checksum     string `string:"Checksum"`
	Parameters   string `string:"Parameters"`
}

type ExtractorsConfig struct {
	Extractors                 []Extractor `yaml:"Extractor"`
	Default                    string      `string:"Default"`
	MetadataExtractorsLocation string      `string:"MetadataExtractorsLocation"`
}
