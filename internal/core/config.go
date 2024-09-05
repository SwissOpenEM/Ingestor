package core

import (
	"fmt"
	"os"
	"path"

	"github.com/paulscherrerinstitute/scicat-cli/cmd/cliutils"
	"github.com/spf13/viper"
)

type Scicat struct {
	Host        string `string:"Host"`
	AccessToken string `string:"AccessToken"`
}

type S3 struct {
	Endpoint string `string:"Endpoint"`
	Bucket   string `string:"Bucket"`
	Location string `string:"Location"`
	User     string `string:"User"`
	Password string `string:"Password"`
	Checksum bool   `bool:"Checksum"`
}

type Globus struct {
	Endpoint string `string:"Endpoint"`
}

type Transfer struct {
	Method string                `string:"Method"`
	S3     S3                    `mapstructure:"S3"`
	Globus cliutils.GlobusConfig `mapstructure:"cliutils.GlobusConfig"`
}

type Misc struct {
	ConcurrencyLimit int `int:"ConcurrencyLimit"`
}

type Config struct {
	Scicat   Scicat   `mapstructure:"Scicat"`
	Transfer Transfer `mapstructure:"Transfer"`
	Misc     Misc     `mapstructure:"Misc"`
}

func GetConfig() (Config, error) {
	var config Config
	if err := viper.Unmarshal(&config); err != nil {
		fmt.Println(err)
		return config, err
	}
	return config, nil
}

func ReadConfig() error {
	viper.SetConfigName("openem-ingestor-config") // name of config file (without extension)
	viper.SetConfigType("yaml")

	userConfigDir, _ := os.UserConfigDir()
	executablePath, _ := os.Executable()

	// Give priority to the config file found next to the executable
	viper.AddConfigPath(path.Dir(executablePath))
	viper.AddConfigPath(path.Join(userConfigDir, "openem-ingestor"))

	err := viper.ReadInConfig()
	return err
}
