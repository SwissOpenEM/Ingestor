package core

import (
	"fmt"
	"os"
	"path"

	"github.com/SwissOpenEM/Ingestor/internal/metadataextractor"
	"github.com/SwissOpenEM/Ingestor/internal/transfertask"
	"github.com/SwissOpenEM/Ingestor/internal/webserver/wsconfig"
	"github.com/go-playground/validator/v10"
	"github.com/spf13/viper"
)

type ScicatConfig struct {
	Host string `string:"Host" validate:"required,url"`
}

type Config struct {
	Scicat             ScicatConfig                       `mapstructure:"Scicat"`
	Transfer           transfertask.TransferConfig        `mapstructure:"Transfer"`
	WebServer          wsconfig.WebServerConfig           `mapstructure:"WebServer"`
	MetadataExtractors metadataextractor.ExtractorsConfig `mapstructure:"MetadataExtractors"`
}

type ConfigReader struct {
	viperConf *viper.Viper
}

func NewConfigReader() ConfigReader {
	userConfigDir, _ := os.UserConfigDir()
	executablePath, _ := os.Executable()

	// Give priority to the config file found next to the executable
	viperConf := viper.New()
	viperConf.AddConfigPath(path.Dir(executablePath))
	viperConf.AddConfigPath(path.Join(userConfigDir, "openem-ingestor"))
	viperConf.SetConfigType("yaml")
	return ConfigReader{viperConf: viperConf}
}

func (c *ConfigReader) getConfig() (Config, error) {
	var config Config
	if err := c.viperConf.UnmarshalExact(&config); err != nil {
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

func (c *ConfigReader) ReadConfig(configFileName string) (Config, error) {
	c.viperConf.SetConfigName(configFileName) // name of config file (without extension)

	c.viperConf.SetDefault("Scicat.Host", "https://datcat.psi.ch/api/v3")

	c.viperConf.SetDefault("MetadataExtractors.InstallationPath", "./extractors/")
	c.viperConf.SetDefault("MetadataExtractors.SchemasLocation", "./schemas/")
	c.viperConf.SetDefault("MetadataExtractors.DownloadSchemas", true)
	c.viperConf.SetDefault("MetadataExtractors.DownloadMissingExtractors", true)
	c.viperConf.SetDefault("MetadataExtractors.Timeout", "10m")

	c.viperConf.SetDefault("WebServer.Auth.Disable", false)
	c.viperConf.SetDefault("WebServer.Auth.Frontend.Origin", "https://discovery.psi.ch")
	c.viperConf.SetDefault("WebServer.Auth.Frontend.RedirectPath", "/ingestor")
	c.viperConf.SetDefault("WebServer.Auth.OAuth2.ClientID", "ingestor")
	c.viperConf.SetDefault("WebServer.Auth.OAuth2.RedirectURL", "https://keycloak.psi.ch/callback")
	c.viperConf.SetDefault("WebServer.Auth.OAuth2.Scopes", []string{"email"})
	c.viperConf.SetDefault("WebServer.Auth.OIDC.IssuerURL", "http://keycloak.psi.ch/realms/facility")
	c.viperConf.SetDefault("WebServer.Auth.JWT.UseJWKS", true)
	c.viperConf.SetDefault("WebServer.Auth.JWT.JwksURL", "https://keycloak.psi.ch/realms/facility/protocol/openid-connect/certs")
	c.viperConf.SetDefault("WebServer.Auth.JWT.JwksSignatureMethods", []string{"RS256"})

	c.viperConf.SetDefault("WebServer.MetadataExtJobs.ConcurrencyLimit", 10)
	c.viperConf.SetDefault("WebServer.MetadataExtJobs.QueueSize", 200)

	c.viperConf.SetDefault("WebServer.Other.Port", 8888)
	c.viperConf.SetDefault("WebServer.Other.LogLevel", "Info")
	c.viperConf.SetDefault("WebServer.Other.DisableServiceAccountCheck", false)
	c.viperConf.SetDefault("WebServer.Other.GlobalConcurrencyLimit", 64)

	err := c.viperConf.ReadInConfig()
	if err == nil {
		config, err := c.getConfig()
		return config, err
	}
	return Config{}, err
}

func (c *ConfigReader) GetCurrentConfigFilePath() string {
	return c.viperConf.ConfigFileUsed()
}

func (c *ConfigReader) GetFullConfig() map[string]any {
	return c.viperConf.AllSettings()
}

func (c *ConfigReader) SetConfKey(key string, value any) {
	c.viperConf.Set(key, value)
}

func (c *ConfigReader) SaveConfig() error {
	return c.viperConf.WriteConfig()
}
