package datasetaccess

type accessFile struct {
	Path         string   `yaml:"Path"`
	AccessGroups []string `yaml:"AccessGroups"`
}
