package webserver

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/SwissOpenEM/Ingestor/internal/refreshfunctoken"
	"github.com/SwissOpenEM/Ingestor/internal/webserver/randomfuncs"
	"github.com/SwissOpenEM/globus"
	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
	"golang.org/x/oauth2"
)

func loggedIntoGlobus(ctx *gin.Context) bool {
	globusSession := sessions.DefaultMany(ctx, "globus")
	rt, ok1 := globusSession.Get("refresh_token").(string)
	at, ok2 := globusSession.Get("access_token").(string)
	e, ok3 := globusSession.Get("expiry").(string)
	if !(ok1 && ok2 && ok3) || rt == "" || at == "" || e == "" {
		return false
	}
	_, err := time.Parse(time.RFC3339Nano, e)
	if err != nil {
		return false
	}

	return true
}

func globusCallbackRedirect(ctx context.Context, globusAuthConf *oauth2.Config) (GetCallbackResponseObject, error) {
	redirectUrl, err := globusRedirect(ctx, globusAuthConf)
	if err != nil {
		return GetCallback500TextResponse(err.Error()), nil
	}

	return GetCallback302Response{
		Headers: GetCallback302ResponseHeaders{
			Location: redirectUrl,
		},
	}, nil
}

func globusLoginRedirect(ctx context.Context, globusAuthConf *oauth2.Config) (GetLoginResponseObject, error) {
	redirectUrl, err := globusRedirect(ctx, globusAuthConf)
	if err != nil {
		return nil, err
	}

	return GetLogin302Response{
		Headers: GetLogin302ResponseHeaders{
			Location: redirectUrl,
		},
	}, nil
}

func globusRedirect(ctx context.Context, globusAuthConf *oauth2.Config) (string, error) {
	// get sessions
	ginCtx, ok := ctx.(*gin.Context)
	if !ok {
		return "", fmt.Errorf("can't access gin context")
	}
	authSession := sessions.DefaultMany(ginCtx, "auth")

	// generate state, verifier and nonce
	state, err := randomfuncs.GenerateRandomString(16)
	if err != nil {
		return "", fmt.Errorf("can't generate random string: %s", err.Error())
	}
	verifier := oauth2.GenerateVerifier()

	// store state, verifier & nonce in session
	authSession.Options(sessions.Options{
		HttpOnly: true,
		MaxAge:   300,
		Secure:   ginCtx.Request.TLS != nil,
	})
	authSession.Set("state", state)
	authSession.Set("verifier", verifier)
	err = authSession.Save()
	if err != nil {
		return "", fmt.Errorf("can't create auth session cookie: %s", err.Error())
	}

	return globusAuthConf.AuthCodeURL(
		state,
		oauth2.AccessTypeOffline,
		oauth2.S256ChallengeOption(verifier),
	), nil
}

// this is the callback endpoint for handling the globus code exchange
func (i *IngestorWebServerImplemenation) GetGlobusCallback(ctx context.Context, request GetGlobusCallbackRequestObject) (GetGlobusCallbackResponseObject, error) {
	ginCtx, ok := ctx.(*gin.Context)
	if !ok {
		return GetGlobusCallback500TextResponse("can't access context"), nil
	}
	authSession := sessions.DefaultMany(ginCtx, "auth")
	globusSession := sessions.DefaultMany(ginCtx, "globus")

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
		Secure:   ginCtx.Request.TLS != nil,
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

	globusSession.Set("refresh_token", oauthToken.RefreshToken)
	globusSession.Set("access_token", oauthToken.AccessToken)
	globusSession.Set("expiry", oauthToken.Expiry.Format(time.RFC3339Nano))
	globusSession.Options(sessions.Options{
		HttpOnly: true,
		MaxAge:   int(i.sessionDuration),
		Secure:   ginCtx.Request.TLS != nil,
	})
	globusSession.Save()

	return GetGlobusCallback302Response{
		Headers: GetGlobusCallback302ResponseHeaders{
			Location: i.frontend.origin + i.frontend.redirectPath,
		},
	}, nil
}

func globusLogout(ctx *gin.Context, globusConf oauth2.Config) error {
	globusSession := sessions.DefaultMany(ctx, "globus")

	accessToken, ok1 := globusSession.Get("access_token").(string)
	refreshToken, ok2 := globusSession.Get("refresh_token").(string)
	expiryStr, ok3 := globusSession.Get("expiry").(string)
	if !ok1 || !ok2 || !ok3 {
		return fmt.Errorf("session expired")
	}

	expiry, err := time.Parse(time.RFC3339Nano, expiryStr)
	if err != nil {
		return fmt.Errorf("can't parse time: %s", err.Error())
	}
	_ = expiry

	// attempt to invalidate both before returning any errors
	var revokeErrs [2]error

	if globusConf.ClientSecret != "" {
		revokeErrs[0] = globusRevokeToken(globusConf.ClientID, globusConf.ClientSecret, accessToken)
		revokeErrs[1] = globusRevokeToken(globusConf.ClientID, globusConf.ClientSecret, refreshToken)
	}

	globusSession.Delete("access_token")
	globusSession.Delete("refresh_token")
	globusSession.Delete("expiry")
	globusSession.Options(sessions.Options{
		HttpOnly: true,
		Secure:   ctx.Request.TLS != nil,
		MaxAge:   -1,
	})
	globusSession.Save()

	return errors.Join(revokeErrs[0], revokeErrs[1]) // return potential revocation errors
}

func globusRevokeToken(clientId string, clientSecret string, token string) error {
	// warning: this only works with confidential clients, clientSecret *cannot* be empty
	if clientSecret == "" {
		return fmt.Errorf("client secret was not set")
	}

	client := http.DefaultClient
	req, err := http.NewRequest("POST", "https://auth.globus.org/v2/oauth2/token/revoke", nil)
	if err != nil {
		return err
	}

	req.Header.Add("Authorization", "Basic "+base64.StdEncoding.EncodeToString([]byte(clientId+":"+clientSecret)))

	q := req.URL.Query()
	q.Set("token", token)
	req.URL.RawQuery = q.Encode()

	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode == 200 {
		return nil // invalidation succeeded
	}

	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("got status code %d, but failed to read body: %s", resp.StatusCode, err.Error())
	}

	return fmt.Errorf("got status code %d - '%s'", resp.StatusCode, string(b))
}

func (i *IngestorWebServerImplemenation) globusRefreshToken(ctx *gin.Context, refreshToken string) (string, string, time.Time, error) {
	type tokenResponse struct {
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token"`
		ExpiresIn    int    `json:"expires_in"`
		TokenType    string `json:"token_type"`
	}

	client := i.globusAuthConf.Client(context.Background(), nil)

	req, err := http.NewRequest("POST", "https://auth.globus.org/v2/oauth2/token", nil)
	if err != nil {
		return "", "", time.Time{}, err
	}
	q := req.URL.Query()
	q.Add("grant_type", "refresh_token")
	q.Add("token", refreshToken)
	req.URL.RawQuery = q.Encode()

	timeAtRequest := time.Now()
	resp, err := client.Do(req)
	if err != nil {
		return "", "", time.Time{}, err
	}
	defer resp.Body.Close()

	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", "", time.Time{}, fmt.Errorf("status Code %d received, but can't decode body: %s", resp.StatusCode, err.Error())
	}

	if resp.StatusCode != 200 {
		return "", "", time.Time{}, fmt.Errorf("status Code %d received: %s", resp.StatusCode, string(b))
	}

	var t tokenResponse
	err = json.Unmarshal(b, &t)
	if err != nil {
		return "", "", time.Time{}, err
	}

	expiry := timeAtRequest.Add(time.Duration(t.ExpiresIn) * time.Second)

	// update context cookies if context still exists
	if ctx.Err() == nil {
		globusSession := sessions.DefaultMany(ctx, "globus")
		globusSession.Set("refresh_token", t.RefreshToken)
		globusSession.Set("access_token", t.AccessToken)
		globusSession.Set("expiry", expiry.Format(time.RFC3339Nano))
		globusSession.Save()
	}

	return t.AccessToken, t.RefreshToken, expiry, nil
}

func (i *IngestorWebServerImplemenation) globusGetClientFromSession(ctx context.Context) (*globus.GlobusClient, error) {
	ginCtx := ctx.(*gin.Context)
	globusSession := sessions.DefaultMany(ginCtx, "globus")
	refreshToken, ok1 := globusSession.Get("refresh_token").(string)
	accessToken, ok2 := globusSession.Get("access_token").(string)
	expiryStr, ok3 := globusSession.Get("expiry").(string)

	if !(ok1 && ok2 && ok3) {
		return nil, fmt.Errorf("globus session has expired")
	}

	expiry, err := time.Parse(time.RFC3339Nano, expiryStr)
	if err != nil {
		return nil, err
	}

	ts := refreshfunctoken.NewTokenSource(ginCtx, accessToken, refreshToken, expiry, i.globusRefreshToken)

	client := globus.HttpClientToGlobusClient(oauth2.NewClient(ctx, ts))
	return &client, nil
}
