package core

import (
	"fmt"
	"log/slog"
	"os"
	"path"

	"github.com/SwissOpenEM/Ingestor/internal/metadataextractor"
	"github.com/SwissOpenEM/Ingestor/internal/task"
	"github.com/go-playground/validator/v10"
	"github.com/spf13/viper"
	"golang.org/x/oauth2"
)

type ScicatConfig struct {
	Host        string `string:"Host" validate:"required,url"`
	AccessToken string `string:"AccessToken"`
}

type OAuth2Conf struct {
	ClientID     string          // OAuth client id (this app)
	ClientSecret string          // OAuth2 secret (associated with ClientID)
	Endpoint     oauth2.Endpoint // Oauth2 endpoint params
	RedirectURL  string          // where should the OAuth2 provider return the user to
	Scopes       []string        // list of scopes to ask for from the OAuth2 provider
}

type OIDCConf struct {
	IssuerURL string

	// the ones below are only needed if the OIDC discovery mechanism doesn't work
	AuthURL     string
	TokenURL    string
	UserInfoURL string
	Algorithms  []string
}

type JWTConf struct {
	ClientID string // Client ID of this server in IdP (Keycloak)
	UseJWKS  bool
	// used when UseJWKS is set to true
	JwksURL              string
	JwksSignatureMethods []string
	// used when UseJWKS is set to false
	Key           string // public key in case of asymmetric method, otherwise common secret (HMAC)
	KeySignMethod string // can be "HS#", "RS#", "EC#", "EdDSA" (where # can be 256, 384, 512)
}

type RBACConf struct {
	AdminRole             string
	CreateModifyTasksRole string
	ViewTasksRole         string
}

type AuthConf struct {
	Disable         bool
	SessionDuration uint // duration of a user session before it expires (by default never)
	JWTConf         `mapstructure:"JWT"`
	RBACConf        `mapstructure:"RBAC"`
	OAuth2Conf      `mapstructure:"OAuth2"`
	OIDCConf        `mapstructure:"OIDC"`
}

type MiscConfig struct {
	ConcurrencyLimit int `int:"ConcurrencyLimit" validate:"gte=0"`
	Port             int `int:"Port" validate:"required,gte=0"`
}

type Config struct {
	Auth               AuthConf                           `mapstructure:"Auth"`
	Scicat             ScicatConfig                       `mapstructure:"Scicat"`
	Transfer           task.TransferConfig                `mapstructure:"Transfer"`
	Misc               MiscConfig                         `mapstructure:"Misc"`
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
		slog.Error("Configuration validation failed:", "error", err.Error())
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

	viper.SetDefault("Misc.Port", 8888)

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
	return Config{}, nil
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
