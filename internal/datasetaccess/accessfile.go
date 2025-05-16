package datasetaccess

type accessFile struct {
	Path          string   `yaml:"Path"`
	AllowedGroups []string `yaml:"AllowedGroups"`
	BlockedGroups []string `yaml:"BlockedGroups"`
}
