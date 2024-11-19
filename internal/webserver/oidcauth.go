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
	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
	"golang.org/x/oauth2"
)

type tokenClaims struct {
	Scope string `json:"scope"`
}

type loginFlowCookie struct {
	State    []byte `json:"state"`
	Verifier []byte `json:"verifier"`
}

var oidcVerifier *oidc.IDTokenVerifier

func oidcAuthFunc(ctx context.Context, input *openapi3filter.AuthenticationInput) error {
	// find user scopes (currently disabled, could be a nice fallback solution later)
	/* token := input.RequestValidationInput.Request.Header.Get("Authorization")
	if token == "" {
		return errors.New("authorization header required")
	}*/

	ginCtx := ctx.(*gin.Context)
	authSession := sessions.DefaultMany(ginCtx, "auth")
	tokenSource, ok := authSession.Get("auth_token_source").(oauth2.TokenSource)
	if !ok {
		return errors.New("user is not logged in")
	}

	oauthToken, err := tokenSource.Token()
	if err != nil {
		return fmt.Errorf("can't obtain token: %s", err.Error())
	}

	// get id token (not sure if needed here?)
	rawIDToken, ok := oauthToken.Extra("id_token").(string)
	if !ok {
		return errors.New("id token is not part of the oauth token")
	}

	idToken, err := oidcVerifier.Verify(ctx, rawIDToken)
	if err != nil {
		return fmt.Errorf("can't verify token: %s", err.Error())
	}

	var claims tokenClaims
	if err := idToken.Claims(&claims); err != nil {
		return fmt.Errorf("can't extract claims: %s", err.Error())
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

func generateRandomByteSlice(len uint) ([]byte, error) {
	b := make([]byte, len)
	_, err := rand.Read(b)
	if err != nil {
		return []byte{}, err
	}
	return b, nil
}

func generateRandomString(len uint) (string, error) {
	b, err := generateRandomByteSlice(len)
	return base64.URLEncoding.EncodeToString(b), err
}
