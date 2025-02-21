package wsconfig

type OAuth2Conf struct {
	ClientID     string   `validate:"required"` // OAuth client id (this app)
	ClientSecret string   // OAuth2 secret (associated with ClientID, optional)
	RedirectURL  string   `validate:"required"` // where should the OAuth2 provider return the user to
	Scopes       []string // list of scopes to ask for from the OAuth2 provider
}

type OIDCConf struct {
	IssuerURL string `validate:"url,omitempty"`

	// the ones below are only needed if the OIDC discovery mechanism doesn't work
	AuthURL     string `validate:"omitempty,url"`
	TokenURL    string `validate:"omitempty,url"`
	UserInfoURL string `validate:"omitempty,url"`
	Algorithms  []string
}

// config for JWT token signature check
type JWTConf struct {
	UseJWKS bool `bool:"UseJWKS" validate:"required"`
	// used when UseJWKS is set to true
	JwksURL              string   `validate:"required_if=UseJWKS true,url,omitempty"`
	JwksSignatureMethods []string `validate:"required_if=UseJWKS true,omitempty"`
	// used when UseJWKS is set to false
	Key           string `validate:"required_if=UseJWKS false,omitempty"`                                                                   // public key in case of asymmetric method, otherwise common secret (HMAC)
	KeySignMethod string `validate:"required_if=UseJWKS false,omitempty,oneof=HS256 HS384 HS512 RS256 RS384 RS512 EC256 EC384 EC512 EdDSA"` // can be "HS#", "RS#", "EC#", "EdDSA" (where # can be 256, 384, 512)
}

// for configuring various role category names (could be different per facility)
type RBACConf struct {
	AdminRole             string `validate:"required"`
	CreateModifyTasksRole string `validate:"required"`
	ViewTasksRole         string `validate:"required"`
}

type FrontendConf struct {
	Origin       string `validate:"required"`
	RedirectPath string
}

// full authentication config
type AuthConf struct {
	Disable         bool `bool:"Disable"`
	SessionDuration uint // duration of a user session before it expires (by default never)
	FrontendConf    `mapstructure:"Frontend" validate:"required_if=Disable false,omitempty"`
	OAuth2Conf      `mapstructure:"OAuth2" validate:"required_if=Disable false,omitempty"`
	OIDCConf        `mapstructure:"OIDC" validate:"required_if=Disable false,omitempty"`
	JWTConf         `mapstructure:"JWT" validate:"required_if=Disable false,omitempty"`
	RBACConf        `mapstructure:"RBAC" validate:"required_if=Disable false,omitempty"`
}

type PathsConf struct {
	CollectionLocation      string `validate:"required"`
	ExtractorOutputLocation string
}

type MetadataExtJobsConf struct {
	ConcurrencyLimit int `validate:"required,min=1"`
	QueueSize        int `validate:"min=0"`
}

type OtherConf struct {
	Port int `int:"Port" validate:"required,gte=0"`
}

type WebServerConfig struct {
	AuthConf            `mapstructure:"Auth"`
	PathsConf           `mapstructure:"Paths"`
	MetadataExtJobsConf `mapstructure:"MetadataExtJobs"`
	OtherConf           `mapstructure:"Other"`
}
