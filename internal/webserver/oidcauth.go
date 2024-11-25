package webserver

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/MicahParks/keyfunc/v3"
	"github.com/SwissOpenEM/Ingestor/internal/core"
	"github.com/getkin/kin-openapi/openapi3filter"
	"github.com/golang-jwt/jwt/v5"
)

type tokenClaims struct {
	Scope string `json:"scope"`
}

func initKeyfunc(authConf core.AuthConf) (jwt.Keyfunc, error) {
	if authConf.UseJWKS {
		jwks, err := keyfunc.NewDefault([]string{authConf.JwksURL})
		if err != nil {
			return nil, err
		}
		return jwks.Keyfunc, nil
	} else {
		return func(token *jwt.Token) (interface{}, error) {
			// validate signing algorithm
			if token.Header["alg"] != authConf.PKeySignMethod {
				return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
			}

			// return public key if the algorithm is what is expected
			return jwt.ParseRSAPublicKeyFromPEM([]byte(authConf.PublicKey))
		}, nil
	}
}

func (i *IngestorWebServerImplemenation) oidcAuthFunc(ctx context.Context, input *openapi3filter.AuthenticationInput) error {
	// find user scopes (currently disabled, could be a nice fallback solution later)
	bearer := input.RequestValidationInput.Request.Header.Get("Authorization")
	if bearer == "" {
		return errors.New("user is not logged-in")
	}
	splitToken := strings.Split(bearer, "Bearer ")
	if len(splitToken) != 2 {
		return errors.New("invalid bearer token")
	}

	jwtToken := splitToken[1]

	token, err := jwt.Parse(jwtToken, i.jwtKeyfunc, jwt.WithValidMethods(i.jwtSignMethods))
	if err != nil {
		return err // token is not valid (expired), likely
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return errors.New("can't extract claims")
	}

	fmt.Printf("here's the claims of the token: \n=====\n%v\n=====\n", claims)

	return nil // for now we accept anything that has a valid JWT token
}

func findMissingScopes(desiredScopes []string, actualScopes []string) []string {
	missingScopes := []string{}
	scopeMap := make(map[string]bool)

	for _, scope := range actualScopes {
		scopeMap[scope] = false
	}

	for _, scope := range desiredScopes {
		if _, ok := scopeMap[scope]; !ok {
			missingScopes = append(missingScopes, scope)
		}
	}

	return missingScopes
}

func missingScopesCheck(missingScopes []string, methodName string) error {
	if len(missingScopes) == 0 {
		return nil
	}
	return fmt.Errorf("missing scopes for \"%s\": %v", methodName, missingScopes)
}
