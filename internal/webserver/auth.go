package webserver

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"slices"
	"time"

	"github.com/SwissOpenEM/Ingestor/internal/transfertask"
	"github.com/SwissOpenEM/Ingestor/internal/webserver/globusauth"
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
		Secure:   i.secureCookies || (ginCtx.Request.TLS != nil),
		SameSite: http.SameSiteNoneMode,
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
		Secure:   i.secureCookies || (ginCtx.Request.TLS != nil),
		SameSite: http.SameSiteNoneMode,
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
		Secure:   i.secureCookies || (ginCtx.Request.TLS != nil),
		SameSite: http.SameSiteNoneMode,
	})
	err = userSession.Save()
	if err != nil {
		return GetCallback500TextResponse(fmt.Sprintf("can't set user session: %s", err.Error())), nil
	}

	// globus redirect for logging-in (if using globus)
	if i.taskQueue.GetTransferMethod() == transfertask.TransferGlobus {
		// revoke session with globus, if we have one ongoing
		if globusauth.TestGlobusCookie(ginCtx) {
			_ = globusauth.Logout(ginCtx, *i.globusAuthConf, i.secureCookies) // we don't care if logout fails
		}
		return globusCallbackRedirect(ctx, i.globusAuthConf, i.secureCookies)
	}

	redirectUrl := i.frontend.origin + i.frontend.redirectPath + "?backendUrl=" + url.QueryEscape(i.taskQueue.Config.WebServer.BackendAddress)

	// standard redirect to frontend if there's nothing else to do
	return GetCallback302Response{
		Headers: GetCallback302ResponseHeaders{
			Location: redirectUrl,
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
		Secure:   i.secureCookies || (ginCtx.Request.TLS != nil),
		SameSite: http.SameSiteNoneMode,
		MaxAge:   -1,
	})
	err := userSession.Save()
	if err != nil {
		return GetLogout500TextResponse(err.Error()), nil
	}

	if i.taskQueue.GetTransferMethod() == transfertask.TransferGlobus {
		err = globusauth.Logout(ginCtx, *i.globusAuthConf, i.secureCookies)
		if err != nil {
			return GetLogout500TextResponse(err.Error()), nil
		}
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

// this is the callback endpoint for handling the globus code exchange
func (i *IngestorWebServerImplemenation) GetGlobusCallback(ctx context.Context, request GetGlobusCallbackRequestObject) (GetGlobusCallbackResponseObject, error) {
	ginCtx, ok := ctx.(*gin.Context)
	if !ok {
		return GetGlobusCallback500TextResponse("can't access context"), nil
	}
	authSession := sessions.DefaultMany(ginCtx, "auth")

	state, ok1 := authSession.Get("state").(string)
	verifier, ok2 := authSession.Get("verifier").(string)
	if !ok1 || !ok2 {
		return GetGlobusCallback400TextResponse("auth session has expired or is invalid"), nil
	}

	// delete auth session
	authSession.Delete("state")
	authSession.Delete("verifier")
	authSession.Options(sessions.Options{
		HttpOnly: true,
		Secure:   i.secureCookies || (ginCtx.Request.TLS != nil),
		SameSite: http.SameSiteNoneMode,
		MaxAge:   -1,
	})
	err := authSession.Save()
	if err != nil {
		return GetGlobusCallback500TextResponse(err.Error()), nil
	}

	if request.Params.State != state {
		return GetGlobusCallback400TextResponse("invalid state"), nil
	}

	// exchange authorization code for accessToken
	oauthToken, err := i.globusAuthConf.Exchange(
		ctx,
		request.Params.Code,
		oauth2.AccessTypeOffline,
		oauth2.VerifierOption(verifier),
	)
	if err != nil {
		return GetGlobusCallback400TextResponse(fmt.Sprintf("code exchange failed: %s", err.Error())), nil
	}

	err = globusauth.SetTokenCookie(ginCtx, oauthToken.RefreshToken, oauthToken.AccessToken, oauthToken.Expiry, i.sessionDuration, i.secureCookies)
	if err != nil {
		return GetGlobusCallback400TextResponse(fmt.Sprintf("creating globus session cookie failed: %s", err.Error())), nil
	}

	redirectUrl := i.frontend.origin + i.frontend.redirectPath
	if i.taskQueue.Config.WebServer.BackendAddress != "" {
		redirectUrl += "?backendUrl=" + url.QueryEscape(i.taskQueue.Config.WebServer.BackendAddress) // add connected backend url
	}
	return GetGlobusCallback302Response{
		Headers: GetGlobusCallback302ResponseHeaders{
			Location: redirectUrl,
		},
	}, nil
}

func globusCallbackRedirect(ctx context.Context, globusAuthConf *oauth2.Config, secureCookies bool) (GetCallbackResponseObject, error) {
	redirectUrl, err := globusauth.GetRedirectUrl(ctx, globusAuthConf, secureCookies)
	if err != nil {
		return GetCallback500TextResponse(err.Error()), nil
	}

	return GetCallback302Response{
		Headers: GetCallback302ResponseHeaders{
			Location: redirectUrl,
		},
	}, nil
}
