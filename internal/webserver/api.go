//go:build go1.22

//go:generate go run github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen --config=cfg.yaml ../../api/openapi.yaml
package webserver

import (
	"context"
	"fmt"

	"github.com/SwissOpenEM/Ingestor/internal/core"
	"github.com/SwissOpenEM/Ingestor/internal/metadataextractor"
	"github.com/SwissOpenEM/Ingestor/internal/webserver/wsconfig"
	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/oauth2"
)

var _ StrictServerInterface = (*IngestorWebServerImplemenation)(nil)

type IngestorWebServerImplemenation struct {
	version          string
	taskQueue        *core.TaskQueue
	extractorHandler *metadataextractor.ExtractorHandler
	oauth2Config     *oauth2.Config
	oidcProvider     *oidc.Provider
	oidcVerifier     *oidc.IDTokenVerifier
	jwtKeyfunc       jwt.Keyfunc
	jwtSignMethods   []string
	sessionDuration  uint
	scopeToRoleMap   map[string]string
	pathConfig       wsconfig.WebServerPathsConf
}

func NewIngestorWebServer(version string, taskQueue *core.TaskQueue, extractorHander *metadataextractor.ExtractorHandler, authConf wsconfig.AuthConf, pathConf wsconfig.WebServerPathsConf) (*IngestorWebServerImplemenation, error) {
	oidcProvider, err := oidc.NewProvider(context.Background(), authConf.IssuerURL)
	if err != nil {
		fmt.Println("Warning: OIDC discovery mechanism failed. Falling back to manual OIDC config")
		// fallback provider (this could also be replaced with an error)
		a := &oidc.ProviderConfig{
			IssuerURL:   authConf.IssuerURL,
			AuthURL:     authConf.AuthURL,
			TokenURL:    authConf.TokenURL,
			UserInfoURL: authConf.UserInfoURL,
			Algorithms:  authConf.Algorithms,
		}
		oidcProvider = a.NewProvider(context.Background())
	}
	oidcVerifier := oidcProvider.Verifier(&oidc.Config{ClientID: authConf.OAuth2Conf.ClientID})
	oauthConf := oauth2.Config{
		ClientID:     authConf.OAuth2Conf.ClientID,
		ClientSecret: authConf.ClientSecret,
		Endpoint:     oidcProvider.Endpoint(),
		RedirectURL:  authConf.RedirectURL,
		Scopes:       append([]string{oidc.ScopeOpenID}, authConf.Scopes...),
	}

	keyfunc, err := initKeyfunc(authConf.JWTConf)
	if err != nil {
		return nil, err
	}

	var signMethods []string
	if authConf.UseJWKS {
		signMethods = authConf.JwksSignatureMethods
	} else {
		signMethods = []string{authConf.JWTConf.KeySignMethod}
	}

	scopeToRoleMap, err := createScopeToRoleMap(authConf.RBACConf)
	if err != nil {
		return nil, err
	}

	return &IngestorWebServerImplemenation{
		version:          version,
		taskQueue:        taskQueue,
		extractorHandler: extractorHander,
		oauth2Config:     &oauthConf,
		oidcProvider:     oidcProvider,
		oidcVerifier:     oidcVerifier,
		jwtKeyfunc:       keyfunc,
		jwtSignMethods:   signMethods,
		scopeToRoleMap:   scopeToRoleMap,
		sessionDuration:  authConf.SessionDuration,
		pathConfig:       pathConf,
	}, nil
}
