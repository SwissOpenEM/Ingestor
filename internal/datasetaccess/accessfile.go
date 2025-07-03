package datasetaccess

type accessFile struct {
	HasDatasetFolders bool     `yaml:"HasDatasetFolders"`
	AllowedGroups     []string `yaml:"AllowedGroups"`
	BlockedGroups     []string `yaml:"BlockedGroups"`
}
