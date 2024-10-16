package core

import (
	"fmt"
	"os"
	"path"

	"github.com/SwissOpenEM/Ingestor/internal/task"
	"github.com/spf13/viper"
)

type ScicatConfig struct {
	Host        string `string:"Host"`
	AccessToken string `string:"AccessToken"`
}

type MiscConfig struct {
	ConcurrencyLimit int `int:"ConcurrencyLimit"`
	Port             int `int:"Port"`
}

type Config struct {
	Scicat   ScicatConfig        `mapstructure:"Scicat"`
	Transfer task.TransferConfig `mapstructure:"Transfer"`
	Misc     MiscConfig          `mapstructure:"Misc"`
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

	viper.SetDefault("Misc.Port", 8888)

	userConfigDir, _ := os.UserConfigDir()
	executablePath, _ := os.Executable()

	// Give priority to the config file found next to the executable
	viperConf.AddConfigPath(path.Dir(executablePath))
	viperConf.AddConfigPath(path.Join(userConfigDir, "openem-ingestor"))

	err := viperConf.ReadInConfig()
	return err
}

func GetCurrentConfigFilePath() string {
	return viperConf.ConfigFileUsed()
}

func GetFullConfig() map[string]any {
	return viperConf.AllSettings()
}

func SetConfKey(key string, value any) {
	viperConf.Set(key, value)
}

func SaveConfig() error {
	return viperConf.WriteConfig()
}
