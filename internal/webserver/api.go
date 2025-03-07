//go:build go1.22

//go:generate go run github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen --config=cfg.yaml ../../api/openapi.yaml
package webserver

import (
	"context"
	"fmt"

	"github.com/SwissOpenEM/Ingestor/internal/core"
	"github.com/SwissOpenEM/Ingestor/internal/webserver/metadatatasks"
	"github.com/SwissOpenEM/Ingestor/internal/webserver/wsconfig"
	"github.com/SwissOpenEM/globus"
	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/oauth2"
)

var _ StrictServerInterface = (*IngestorWebServerImplemenation)(nil)

type IngestorWebServerImplemenation struct {
	version         string
	taskQueue       *core.TaskQueue
	metadataExtPool *metadatatasks.MetadataExtractionTaskPool
	oauth2Config    *oauth2.Config
	globusAuthConf  *oauth2.Config
	oidcProvider    *oidc.Provider
	oidcVerifier    *oidc.IDTokenVerifier
	jwtKeyfunc      jwt.Keyfunc
	jwtSignMethods  []string
	sessionDuration uint
	scopeToRoleMap  map[string]string
	pathConfig      wsconfig.PathsConf
	frontend        struct {
		origin       string
		redirectPath string
	}
}

func NewIngestorWebServer(version string, transferQueue *core.TaskQueue, metadataExtPool *metadatatasks.MetadataExtractionTaskPool, serverConf wsconfig.WebServerConfig) (*IngestorWebServerImplemenation, error) {
	oidcProvider, err := oidc.NewProvider(context.Background(), serverConf.IssuerURL)
	if err != nil {
		fmt.Println("Warning: OIDC discovery mechanism failed. Falling back to manual OIDC config")
		// fallback provider (this could also be replaced with an error)
		a := &oidc.ProviderConfig{
			IssuerURL:   serverConf.IssuerURL,
			AuthURL:     serverConf.AuthURL,
			TokenURL:    serverConf.TokenURL,
			UserInfoURL: serverConf.UserInfoURL,
			Algorithms:  serverConf.Algorithms,
		}
		oidcProvider = a.NewProvider(context.Background())
	}
	oidcVerifier := oidcProvider.Verifier(&oidc.Config{ClientID: serverConf.OAuth2Conf.ClientID})
	oauthConf := oauth2.Config{
		ClientID:     serverConf.OAuth2Conf.ClientID,
		ClientSecret: serverConf.ClientSecret,
		Endpoint:     oidcProvider.Endpoint(),
		RedirectURL:  serverConf.RedirectURL,
		Scopes:       append([]string{oidc.ScopeOpenID}, serverConf.Scopes...),
	}

	keyfunc, err := initKeyfunc(serverConf.JWTConf)
	if err != nil {
		return nil, err
	}

	var signMethods []string
	if serverConf.UseJWKS {
		signMethods = serverConf.JwksSignatureMethods
	} else {
		signMethods = []string{serverConf.JWTConf.KeySignMethod}
	}

	scopeToRoleMap, err := createScopeToRoleMap(serverConf.RBACConf)
	if err != nil {
		return nil, err
	}

	globusAuthConf := globus.AuthGenerateOauthClientConfig(
		context.Background(),
		transferQueue.Config.Transfer.Globus.ClientID,
		transferQueue.Config.Transfer.Globus.ClientSecret,
		transferQueue.Config.Transfer.Globus.RedirectURL,
		transferQueue.Config.Transfer.Globus.Scopes,
	)

	return &IngestorWebServerImplemenation{
		version:         version,
		taskQueue:       transferQueue,
		oauth2Config:    &oauthConf,
		globusAuthConf:  &globusAuthConf,
		oidcProvider:    oidcProvider,
		oidcVerifier:    oidcVerifier,
		jwtKeyfunc:      keyfunc,
		jwtSignMethods:  signMethods,
		scopeToRoleMap:  scopeToRoleMap,
		sessionDuration: serverConf.SessionDuration,
		pathConfig:      serverConf.PathsConf,
		metadataExtPool: metadataExtPool,
		frontend: struct {
			origin       string
			redirectPath string
		}{
			origin:       serverConf.FrontendConf.Origin,
			redirectPath: serverConf.FrontendConf.RedirectPath,
		},
	}, nil
}
