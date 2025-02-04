package webserver

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"time"

	"github.com/SwissOpenEM/Ingestor/internal/task"
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
					Location: i.frontend.origin + i.frontend.redirectPath,
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
	ginCtx, ok := ctx.(*gin.Context)
	if !ok {
		return GetCallback500TextResponse("can't access context"), nil
	}
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
	userSession.Set("expires_at", time.Now().Add(time.Second*time.Duration(i.sessionDuration)).Format(time.RFC3339Nano))
	userSession.Set("email", userInfo.Email)
	userSession.Set("profile", userInfo.Profile)
	userSession.Set("subject", userInfo.Subject)
	userSession.Set("roles", claims.GetResourceRolesByKey(i.oauth2Config.ClientID))
	userSession.Set("preferred_username", claims.PreferredUsername)
	userSession.Set("name", claims.Name)
	userSession.Set("family_name", claims.FamilyName)
	userSession.Set("given_name", claims.GivenName)
	userSession.Options(sessions.Options{
		HttpOnly: true,
		MaxAge:   int(i.sessionDuration),
		Secure:   ginCtx.Request.TLS != nil,
	})
	err = userSession.Save()
	if err != nil {
		return GetCallback500TextResponse(fmt.Sprintf("can't set user session: %s", err.Error())), nil
	}

	// globus login (if using globus)
	/*if i.taskQueue.GetTransferMethod() == task.TransferGlobus {
		return globusLoginRedirect(ctx, i.globusAuthConf)
	}*/

	// reply
	return GetCallback302Response{
		Headers: GetCallback302ResponseHeaders{
			Location: i.frontend.origin + i.frontend.redirectPath,
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

	if i.taskQueue.GetTransferMethod() == task.TransferGlobus {
		//
	}

	return GetLogout302Response{GetLogout302ResponseHeaders{
		Location: i.frontend.origin + i.frontend.redirectPath,
	}}, nil
}

func (i *IngestorWebServerImplemenation) GetUserinfo(ctx context.Context, request GetUserinfoRequestObject) (GetUserinfoResponseObject, error) {
	if i.taskQueue.Config.WebServer.AuthConf.Disable {
		return GetUserinfo500TextResponse("auth is disabled"), nil
	}

	ginCtx, ok := ctx.(*gin.Context)
	if !ok {
		return GetUserinfo400TextResponse("can't access context"), nil
	}

	userSession := sessions.DefaultMany(ginCtx, "user")
	expiresAtString, ok1 := userSession.Get("expires_at").(string)
	email, ok2 := userSession.Get("email").(string)
	profile, ok3 := userSession.Get("profile").(string)
	subject, ok4 := userSession.Get("subject").(string)
	roles, ok5 := userSession.Get("roles").([]string)
	preferredUsername, ok6 := userSession.Get("preferred_username").(string)
	name, ok7 := userSession.Get("name").(string)
	familyName, ok8 := userSession.Get("family_name").(string)
	givenName, ok9 := userSession.Get("given_name").(string)

	if !(ok1 && ok2 && ok3 && ok4 && ok5 && ok6 && ok7 && ok8 && ok9) {
		return GetUserinfo200JSONResponse{LoggedIn: false}, nil
	}

	expiresAt, err := time.Parse(time.RFC3339Nano, expiresAtString)
	if err != nil {
		return GetUserinfo500TextResponse("can't parse expiry"), nil
	}

	if expiresAt.Before(time.Now()) {
		return GetUserinfo200JSONResponse{LoggedIn: false}, nil
	}

	strPointerOrNil := func(str *string) *string {
		if str == nil {
			return str
		}
		if *str == "" {
			return nil
		} else {
			return str
		}
	}

	return GetUserinfo200JSONResponse{
		LoggedIn:          true,
		Email:             strPointerOrNil(&email),
		Profile:           strPointerOrNil(&profile),
		Subject:           strPointerOrNil(&subject),
		Roles:             &roles,
		PreferredUsername: strPointerOrNil(&preferredUsername),
		Name:              strPointerOrNil(&name),
		FamilyName:        strPointerOrNil(&familyName),
		GivenName:         strPointerOrNil(&givenName),
		ExpiresAt:         &expiresAt,
	}, nil
}

/*func getGlobusClientFromSession(ctx *gin.Context, conf *oauth2.Config) (globus.GlobusClient, error) {
	globusSession := sessions.DefaultMany(ctx, "globus")
	refreshToken, ok := globusSession.Get("refresh_token").(string)
	if !ok {
		return globus.GlobusClient{}, fmt.Errorf("globus session has expired")
	}

	newToken, err := conf.TokenSource(ctx, &oauth2.Token{RefreshToken: refreshToken}).Token()
	if err != nil {
		return globus.GlobusClient{}, fmt.Errorf("can't refresh token: %s", err.Error())
	}

	globusSession.Set("refresh_token", newToken.RefreshToken)
	globusSession.Save()

	return globus.HttpClientToGlobusClient(conf.Client(ctx, &oauth2.Token{
		TokenType:   newToken.TokenType,
		AccessToken: newToken.AccessToken,
		Expiry:      newToken.Expiry,
		ExpiresIn:   newToken.ExpiresIn,
	})), nil
}*/
