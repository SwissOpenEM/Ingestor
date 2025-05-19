package datasetaccess

type accessFile struct {
	Path              string   `yaml:"Path"`
	HasDatasetFolders bool     `yaml:"HasDatasetFolders"`
	AllowedGroups     []string `yaml:"AllowedGroups"`
	BlockedGroups     []string `yaml:"BlockedGroups"`
}
