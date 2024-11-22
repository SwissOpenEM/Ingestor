package webserver

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/getkin/kin-openapi/openapi3filter"
	"github.com/golang-jwt/jwt/v5"
)

type tokenClaims struct {
	Scope string `json:"scope"`
}

type loginFlowCookie struct {
	State    []byte `json:"state"`
	Verifier []byte `json:"verifier"`
}

var oidcVerifier *oidc.IDTokenVerifier

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
	_ = jwtToken

	token, err := jwt.Parse(jwtToken, func(token *jwt.Token) (interface{}, error) {
		// validate signing algorithm
		if token.Header["alg"] != i.jwtSignatureMethod {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}

		// return public key if the algorithm is what is expected
		return jwt.ParseRSAPublicKeyFromPEM([]byte(i.jwtPublicKey))
	})
	if err != nil {
		return err
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
