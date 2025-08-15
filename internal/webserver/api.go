//go:build go1.22

//go:generate go run github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen --config=cfg.yaml ../../api/openapi.yaml
package webserver

import (
	"context"
	"fmt"

	"github.com/SwissOpenEM/Ingestor/internal/core"
	"github.com/SwissOpenEM/Ingestor/internal/metadataextractor"
	"github.com/SwissOpenEM/Ingestor/internal/webserver/metadatatasks"
	"github.com/SwissOpenEM/Ingestor/internal/webserver/wsconfig"
	"github.com/SwissOpenEM/globus"
	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/oauth2"
)

var _ StrictServerInterface = (*IngestorWebServerImplemenation)(nil)

type IngestorWebServerImplemenation struct {
	version          string
	taskQueue        *core.TaskQueue
	metp             *metadatatasks.MetadataExtractionTaskPool
	extractorHandler *metadataextractor.ExtractorHandler
	oauth2Config     *oauth2.Config
	globusAuthConf   *oauth2.Config
	oidcProvider     *oidc.Provider
	oidcVerifier     *oidc.IDTokenVerifier
	jwtKeyfunc       jwt.Keyfunc
	jwtSignMethods   []string
	sessionDuration  uint
	disableAuth      bool
	scopeToRoleMap   map[string]string
	pathConfig       wsconfig.PathsConf
	secureCookies    bool
	frontend         struct {
		origin       string
		redirectPath string
	}
}

func NewIngestorWebServer(version string, tq *core.TaskQueue, eh *metadataextractor.ExtractorHandler, ws wsconfig.WebServerConfig) (*IngestorWebServerImplemenation, error) {
	oidcProvider, err := oidc.NewProvider(context.Background(), ws.IssuerURL)
	if err != nil {
		fmt.Println("Warning: OIDC discovery mechanism failed. Falling back to manual OIDC config")
		// fallback provider (this could also be replaced with an error)
		a := &oidc.ProviderConfig{
			IssuerURL:   ws.IssuerURL,
			AuthURL:     ws.AuthURL,
			TokenURL:    ws.TokenURL,
			UserInfoURL: ws.UserInfoURL,
			Algorithms:  ws.Algorithms,
		}
		oidcProvider = a.NewProvider(context.Background())
	}
	oidcVerifier := oidcProvider.Verifier(&oidc.Config{ClientID: ws.OAuth2Conf.ClientID})
	oauthConf := oauth2.Config{
		ClientID:     ws.OAuth2Conf.ClientID,
		ClientSecret: ws.ClientSecret,
		Endpoint:     oidcProvider.Endpoint(),
		RedirectURL:  ws.RedirectURL,
		Scopes:       append([]string{oidc.ScopeOpenID}, ws.Scopes...),
	}

	keyfunc, err := initKeyfunc(ws.JWTConf)
	if err != nil {
		return nil, err
	}

	var signMethods []string
	if ws.UseJWKS {
		signMethods = ws.JwksSignatureMethods
	} else {
		signMethods = []string{ws.JWTConf.KeySignMethod}
	}

	scopeToRoleMap, err := createScopeToRoleMap(ws.RBACConf)
	if err != nil {
		return nil, err
	}

	metadataTaskPool := metadatatasks.NewTaskPool(ws.QueueSize, ws.ConcurrencyLimit, eh)

	globusAuthConf := globus.AuthGenerateOauthClientConfig(
		context.Background(),
		tq.Config.Transfer.Globus.ClientID,
		tq.Config.Transfer.Globus.ClientSecret,
		tq.Config.Transfer.Globus.RedirectURL,
		tq.Config.Transfer.Globus.Scopes,
	)

	if tq.ServiceUser == nil && !ws.DisableServiceAccountCheck {
		panic(fmt.Errorf("no service account was set. Set INGESTOR_SERVICE_USER_NAME and INGESTOR_SERVICE_USER_PASSWORD environment variables."))
	}

	return &IngestorWebServerImplemenation{
		version:          version,
		taskQueue:        tq,
		extractorHandler: eh,
		oauth2Config:     &oauthConf,
		globusAuthConf:   &globusAuthConf,
		oidcProvider:     oidcProvider,
		oidcVerifier:     oidcVerifier,
		jwtKeyfunc:       keyfunc,
		jwtSignMethods:   signMethods,
		scopeToRoleMap:   scopeToRoleMap,
		sessionDuration:  ws.SessionDuration,
		disableAuth:      ws.AuthConf.Disable,
		pathConfig:       ws.PathsConf,
		secureCookies:    ws.SecureCookies,
		metp:             metadataTaskPool,
		frontend: struct {
			origin       string
			redirectPath string
		}{
			origin:       ws.FrontendConf.Origin,
			redirectPath: ws.FrontendConf.RedirectPath,
		},
	}, nil
}
