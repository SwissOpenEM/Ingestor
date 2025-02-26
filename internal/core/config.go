package core

import (
	"fmt"
	"os"
	"path"

	"github.com/SwissOpenEM/Ingestor/internal/metadataextractor"
	"github.com/SwissOpenEM/Ingestor/internal/task"
	"github.com/SwissOpenEM/Ingestor/internal/webserver/wsconfig"
	"github.com/go-playground/validator/v10"
	"github.com/spf13/viper"
)

type ScicatConfig struct {
	Host string `string:"Host" validate:"required,url"`
}

type Config struct {
	Scicat             ScicatConfig                       `mapstructure:"Scicat"`
	Transfer           task.TransferConfig                `mapstructure:"Transfer"`
	WebServer          wsconfig.WebServerConfig           `mapstructure:"WebServer"`
	MetadataExtractors metadataextractor.ExtractorsConfig `mapstructure:"MetadataExtractors"`
}

var viperConf *viper.Viper = viper.New()

func getConfig() (Config, error) {
	var config Config
	if err := viperConf.UnmarshalExact(&config); err != nil {
		fmt.Println(err)
		return config, err
	}

	validate := validator.New(validator.WithRequiredStructEnabled())
	err := validate.Struct(&config)
	if err != nil {
		return config, err
	}

	return config, nil
}

func DefaultConfigFileName() string {
	return "openem-ingestor-config"
}

func ReadConfig(configFileName string) (Config, error) {
	viperConf.SetConfigName(configFileName) // name of config file (without extension)
	viperConf.SetConfigType("yaml")

	viper.SetDefault("WebServer.Port", 8888)

	userConfigDir, _ := os.UserConfigDir()
	executablePath, _ := os.Executable()

	// Give priority to the config file found next to the executable
	viperConf.AddConfigPath(path.Dir(executablePath))
	viperConf.AddConfigPath(path.Join(userConfigDir, "openem-ingestor"))

	err := viperConf.ReadInConfig()
	if err == nil {
		config, err := getConfig()
		return config, err
	}
	return Config{}, err
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
