package webserver

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"time"

	"github.com/SwissOpenEM/Ingestor/internal/webserver/randomfuncs"
	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
	"golang.org/x/oauth2"
)

func (i *IngestorWebServerImplemenation) GetLogin(ctx context.Context, request GetLoginRequestObject) (GetLoginResponseObject, error) {
	// auth code flow

	// get sessions
	ginCtx, ok := ctx.(*gin.Context)
	if !ok {
		return GetLogin302Response{}, errors.New("CANT CONVERT")
	}
	authSession := sessions.DefaultMany(ginCtx, "auth")
	userSession := sessions.DefaultMany(ginCtx, "user")

	// check if already logged-in
	if val, ok := userSession.Get("expires_at").(string); ok {
		expiry, _ := time.Parse(time.RFC3339Nano, val)
		if time.Now().Before(expiry) {
			return GetLogin302Response{
				Headers: GetLogin302ResponseHeaders{
					Location: "/",
				},
			}, nil
		}
	}

	// generate state, verifier and nonce
	state, err := randomfuncs.GenerateRandomString(16)
	if err != nil {
		return GetLogin302Response{}, err
	}
	verifier := oauth2.GenerateVerifier()
	nonce, err := randomfuncs.GenerateRandomString(32)
	if err != nil {
		return GetLogin302Response{}, err
	}

	// store state, verifier & nonce in session
	authSession.Options(sessions.Options{
		HttpOnly: true,
		MaxAge:   300,
		Secure:   ginCtx.Request.TLS != nil,
	})
	authSession.Set("state", state)
	authSession.Set("verifier", verifier)
	authSession.Set("nonce", nonce)
	err = authSession.Save()
	if err != nil {
		return GetLogin302Response{}, err
	}

	// redirect to login page
	return GetLogin302Response{
		Headers: GetLogin302ResponseHeaders{
			Location: i.oauth2Config.AuthCodeURL(
				state,
				oauth2.AccessTypeOffline,
				oauth2.S256ChallengeOption(verifier),
				oidc.Nonce(nonce),
			),
		},
	}, nil
}

func (i *IngestorWebServerImplemenation) GetCallback(ctx context.Context, request GetCallbackRequestObject) (GetCallbackResponseObject, error) {
	// get sessions
	ginCtx := ctx.(*gin.Context)
	authSession := sessions.DefaultMany(ginCtx, "auth")
	userSession := sessions.DefaultMany(ginCtx, "user")

	// get auth values
	state, ok1 := authSession.Get("state").(string)
	verifier, ok2 := authSession.Get("verifier").(string)
	nonce, ok3 := authSession.Get("nonce").(string)
	if !(ok1 && ok2 && ok3) {
		return GetCallback400TextResponse("auth session has expired or is invalid"), nil
	}

	// delete auth session
	authSession.Delete("state")
	authSession.Delete("verifier")
	authSession.Delete("nonce")
	authSession.Options(sessions.Options{
		HttpOnly: true,
		Secure:   ginCtx.Request.TLS != nil,
		MaxAge:   -1,
	})
	err := authSession.Save()
	if err != nil {
		return GetCallback500TextResponse(err.Error()), nil
	}

	// verify state (CSRF protection)
	if request.Params.State != state {
		return GetCallback400TextResponse("invalid state"), nil
	}

	// exchange authorization code for accessToken
	oauthToken, err := i.oauth2Config.Exchange(
		ctx,
		request.Params.Code,
		oauth2.AccessTypeOffline,
		oauth2.VerifierOption(verifier),
	)
	if err != nil {
		return GetCallback400TextResponse(fmt.Sprintf("code exchange failed: %s", err.Error())), nil
	}

	// create token source
	tokenSource := i.oauth2Config.TokenSource(ctx, oauthToken)

	// userInfo
	userInfo, err := i.oidcProvider.UserInfo(ctx, tokenSource)
	if err != nil {
		return GetCallback500TextResponse(err.Error()), nil
	}

	// get id token (not sure if needed here?)
	rawIdToken, ok := oauthToken.Extra("id_token").(string)
	if !ok {
		return GetCallback400TextResponse("'id_token' field was not found in oauth2 token"), nil
	}
	idToken, err := i.oidcVerifier.Verify(ctx, rawIdToken)
	if err != nil {
		return GetCallback400TextResponse(fmt.Sprintf("idToken verification failed: %s", err.Error())), nil
	}
	if idToken.Nonce != nonce {
		return GetCallback400TextResponse("nonce did not match"), nil
	}

	claims, err := parseKeycloakJWTToken(oauthToken.AccessToken, i.jwtKeyfunc, i.jwtSignMethods)
	if err != nil {
		return GetCallback500TextResponse(fmt.Sprintf("can't parse jwt token: %s", err.Error())), nil
	}

	// check if AZP is the ClientID
	if claims.AuthorizedParty != i.oauth2Config.ClientID {
		// fallback: check whether the audience contains the client id
		if !slices.Contains([]string(claims.Audience), i.oauth2Config.ClientID) {
			return GetCallback500TextResponse("jwt: the token is not intended for this client"), nil
		}
	}

	// set user session cookie
	userSession.Options(sessions.Options{
		HttpOnly: true,
		MaxAge:   int(i.sessionDuration),
	})
	userSession.Set("expires_at", time.Now().Add(time.Second*time.Duration(i.sessionDuration)).Format(time.RFC3339Nano))
	userSession.Set("email", userInfo.Email)
	userSession.Set("profile", userInfo.Profile)
	userSession.Set("subject", userInfo.Subject)
	userSession.Set("roles", claims.GetResourceRolesByKey(i.oauth2Config.ClientID))
	userSession.Options(sessions.Options{
		HttpOnly: true,
		MaxAge:   int(i.sessionDuration),
		Secure:   ginCtx.Request.TLS != nil,
	})
	err = userSession.Save()
	if err != nil {
		return GetCallback500TextResponse(fmt.Sprintf("can't set user session: %s", err.Error())), nil
	}
	fmt.Printf("access token: \"%s\"\n", oauthToken.AccessToken)

	// reply
	return GetCallback302Response{
		Headers: GetCallback302ResponseHeaders{
			Location: i.frontendUrl,
		},
	}, nil
}

func (i *IngestorWebServerImplemenation) GetLogout(ctx context.Context, request GetLogoutRequestObject) (GetLogoutResponseObject, error) {
	ginCtx, ok := ctx.(*gin.Context)
	if !ok {
		return GetLogout500TextResponse("can't access context"), nil
	}

	// expire session data
	userSession := sessions.DefaultMany(ginCtx, "user")
	userSession.Options(sessions.Options{
		HttpOnly: true,
		Secure:   ginCtx.Request.TLS != nil,
		MaxAge:   -1,
	})
	err := userSession.Save()
	if err != nil {
		return GetLogout500TextResponse(err.Error()), nil
	}

	return GetLogout302Response{GetLogout302ResponseHeaders{
		Location: i.frontendUrl,
	}}, nil
}
