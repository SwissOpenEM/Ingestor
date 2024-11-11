package webserver

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"

	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/getkin/kin-openapi/openapi3filter"
)

type tokenClaims struct {
	Scope string `json:"scope"`
}

var oidcVerifier *oidc.IDTokenVerifier

func oidcAuthFunc(ctx context.Context, input *openapi3filter.AuthenticationInput) error {
	// find user scopes
	token := input.RequestValidationInput.Request.Header.Get("Authorization")
	if token == "" {
		return errors.New("authorization header required")
	}

	idToken, err := oidcVerifier.Verify(context.Background(), token)
	if err != nil {
		return errors.New("invalid token")
	}

	var claims tokenClaims
	if err := idToken.Claims(&claims); err != nil {
		return err
	}

	scopes := strings.Split(claims.Scope, " ")

	// check scopes
	missingScopes := findMissingScopes(input.Scopes, scopes)
	reqPath := input.RequestValidationInput.Request.URL.Path
	reqMethod := input.RequestValidationInput.Request.Method
	return missingScopesCheck(missingScopes, fmt.Sprintf("%s %s", reqMethod, reqPath))
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

func generateState() (string, error) {
	b := make([]byte, 16)
	_, err := rand.Read(b)
	if err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(b), nil
}
