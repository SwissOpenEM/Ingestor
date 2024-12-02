package wsauthconfig

type OAuth2Conf struct {
	ClientID     string   // OAuth client id (this app)
	ClientSecret string   // OAuth2 secret (associated with ClientID, optional)
	RedirectURL  string   // where should the OAuth2 provider return the user to
	Scopes       []string // list of scopes to ask for from the OAuth2 provider
}

type OIDCConf struct {
	IssuerURL string

	// the ones below are only needed if the OIDC discovery mechanism doesn't work
	AuthURL     string
	TokenURL    string
	UserInfoURL string
	Algorithms  []string
}

// config for JWT token signature check
type JWTConf struct {
	UseJWKS bool
	// used when UseJWKS is set to true
	JwksURL              string
	JwksSignatureMethods []string
	// used when UseJWKS is set to false
	Key           string // public key in case of asymmetric method, otherwise common secret (HMAC)
	KeySignMethod string // can be "HS#", "RS#", "EC#", "EdDSA" (where # can be 256, 384, 512)
}

// for configuring various role category names (could be different per facility)
type RBACConf struct {
	AdminRole             string
	CreateModifyTasksRole string
	ViewTasksRole         string
}

// full authentication config
type AuthConf struct {
	Disable         bool
	SessionDuration uint // duration of a user session before it expires (by default never)
	OAuth2Conf      `mapstructure:"OAuth2"`
	OIDCConf        `mapstructure:"OIDC"`
	JWTConf         `mapstructure:"JWT"`
	RBACConf        `mapstructure:"RBAC"`
}
