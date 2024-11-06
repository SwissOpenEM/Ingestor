package webserver

import (
	"context"
	"fmt"

	"github.com/getkin/kin-openapi/openapi3filter"
)

func oidcAuthFunc(ctx context.Context, input *openapi3filter.AuthenticationInput) error {
	reqPath := input.RequestValidationInput.Request.URL.Path
	reqMethod := input.RequestValidationInput.Request.Method

	// find user scopes
	userScopes := []string{}

	missingScopes := findMissingScopes(input.Scopes, userScopes)
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
