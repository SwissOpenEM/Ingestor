package core

import (
	"fmt"
	"os"
	"path"

	"github.com/spf13/viper"
)

type ScicatConfig struct {
	Host        string `string:"Host"`
	AccessToken string `string:"AccessToken"`
}

type S3TransferConfig struct {
	Endpoint string `string:"Endpoint"`
	Bucket   string `string:"Bucket"`
	Location string `string:"Location"`
	User     string `string:"User"`
	Password string `string:"Password"`
	Checksum bool   `bool:"Checksum"`
}

type GlobusTransferConfig struct {
	ClientID              string   `yaml:"client-id"`
	ClientSecret          string   `yaml:"client-secret,omitempty"`
	RedirectURL           string   `yaml:"redirect-url"`
	Scopes                []string `yaml:"scopes,omitempty"`
	SourceCollection      string   `yaml:"source-collection"`
	SourcePrefixPath      string   `yaml:"source-prefix-path,omitempty"`
	DestinationCollection string   `yaml:"destination-collection"`
	DestinationPrefixPath string   `yaml:"destination-prefix-path,omitempty"`
	RenewalToken          string   `yaml:"renewal-token,omitempty"`
}

type TransferConfig struct {
	Method string               `string:"Method"`
	S3     S3TransferConfig     `mapstructure:"S3"`
	Globus GlobusTransferConfig `mapstructure:"cliutils.GlobusConfig"`
}

type MiscConfig struct {
	ConcurrencyLimit int `int:"ConcurrencyLimit"`
}

type Config struct {
	Scicat   ScicatConfig   `mapstructure:"Scicat"`
	Transfer TransferConfig `mapstructure:"Transfer"`
	Misc     MiscConfig     `mapstructure:"Misc"`
}

var viperConf *viper.Viper = viper.New()

func GetConfig() (Config, error) {
	var config Config
	if err := viperConf.Unmarshal(&config); err != nil {
		fmt.Println(err)
		return config, err
	}
	return config, nil
}

func ReadConfig() error {
	viperConf.SetConfigName("openem-ingestor-config") // name of config file (without extension)
	viperConf.SetConfigType("yaml")

	userConfigDir, _ := os.UserConfigDir()
	executablePath, _ := os.Executable()

	// Give priority to the config file found next to the executable
	viperConf.AddConfigPath(path.Dir(executablePath))
	viperConf.AddConfigPath(path.Join(userConfigDir, "openem-ingestor"))

	err := viperConf.ReadInConfig()
	return err
}

func SetConfKey(key string, value any) {
	viperConf.Set(key, value)
}

func SaveConfig() error {
	return viperConf.WriteConfig()
}
