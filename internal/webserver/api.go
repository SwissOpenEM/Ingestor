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
	oidcProvider     *oidc.Provider
	oidcVerifier     *oidc.IDTokenVerifier
	jwtKeyfunc       jwt.Keyfunc
	jwtSignMethods   []string
	sessionDuration  uint
	scopeToRoleMap   map[string]string
	pathConfig       wsconfig.WebServerPathsConf
}

func NewIngestorWebServer(version string, tq *core.TaskQueue, eh *metadataextractor.ExtractorHandler, ac wsconfig.AuthConf, pc wsconfig.WebServerPathsConf) (*IngestorWebServerImplemenation, error) {
	oidcProvider, err := oidc.NewProvider(context.Background(), ac.IssuerURL)
	if err != nil {
		fmt.Println("Warning: OIDC discovery mechanism failed. Falling back to manual OIDC config")
		// fallback provider (this could also be replaced with an error)
		a := &oidc.ProviderConfig{
			IssuerURL:   ac.IssuerURL,
			AuthURL:     ac.AuthURL,
			TokenURL:    ac.TokenURL,
			UserInfoURL: ac.UserInfoURL,
			Algorithms:  ac.Algorithms,
		}
		oidcProvider = a.NewProvider(context.Background())
	}
	oidcVerifier := oidcProvider.Verifier(&oidc.Config{ClientID: ac.OAuth2Conf.ClientID})
	oauthConf := oauth2.Config{
		ClientID:     ac.OAuth2Conf.ClientID,
		ClientSecret: ac.ClientSecret,
		Endpoint:     oidcProvider.Endpoint(),
		RedirectURL:  ac.RedirectURL,
		Scopes:       append([]string{oidc.ScopeOpenID}, ac.Scopes...),
	}

	keyfunc, err := initKeyfunc(ac.JWTConf)
	if err != nil {
		return nil, err
	}

	var signMethods []string
	if ac.UseJWKS {
		signMethods = ac.JwksSignatureMethods
	} else {
		signMethods = []string{ac.JWTConf.KeySignMethod}
	}

	scopeToRoleMap, err := createScopeToRoleMap(ac.RBACConf)
	if err != nil {
		return nil, err
	}

	metp := metadatatasks.NewTaskPool(100, 10, eh)

	return &IngestorWebServerImplemenation{
		version:          version,
		taskQueue:        tq,
		extractorHandler: eh,
		oauth2Config:     &oauthConf,
		oidcProvider:     oidcProvider,
		oidcVerifier:     oidcVerifier,
		jwtKeyfunc:       keyfunc,
		jwtSignMethods:   signMethods,
		scopeToRoleMap:   scopeToRoleMap,
		sessionDuration:  ac.SessionDuration,
		pathConfig:       pc,
		metp:             metp,
	}, nil
}
