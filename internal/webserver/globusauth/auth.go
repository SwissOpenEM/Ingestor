package globusauth

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

func GetRedirectUrl(ctx context.Context, globusAuthConf *oauth2.Config) (string, error) {
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

func Logout(ctx *gin.Context, globusConf oauth2.Config) error {
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
	revokeErrs[0] = revokeToken(globusConf.ClientID, globusConf.ClientSecret, accessToken)
	revokeErrs[1] = revokeToken(globusConf.ClientID, globusConf.ClientSecret, refreshToken)

	DeleteTokenCookie(ctx)

	return errors.Join(revokeErrs[0], revokeErrs[1]) // return potential revocation errors
}

func GetClientFromSession(ctx context.Context, globusConfig *oauth2.Config, sessionDuration uint) (*globus.GlobusClient, error) {
	ginCtx := ctx.(*gin.Context)

	refreshToken, accessToken, expiry, err := GetTokensFromCookie(ginCtx)
	if err != nil {
		return nil, err
	}

	ts := refreshfunctoken.NewTokenSource(ginCtx, globusConfig, accessToken, refreshToken, expiry, sessionDuration, getNewTokens)

	client := globus.HttpClientToGlobusClient(oauth2.NewClient(ctx, ts))
	return &client, nil
}

func getNewTokens(ctx *gin.Context, globusConf *oauth2.Config, refreshToken string, sessionDuration uint) (string, string, time.Time, error) {
	type tokenResponse struct {
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token"`
		ExpiresIn    int    `json:"expires_in"`
		TokenType    string `json:"token_type"`
	}

	client := globusConf.Client(context.Background(), nil)

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
		SetTokenCookie(ctx, t.RefreshToken, t.AccessToken, expiry, sessionDuration)
	}

	return t.AccessToken, t.RefreshToken, expiry, nil
}

func revokeToken(clientId string, clientSecret string, token string) error {
	client := http.DefaultClient
	req, err := http.NewRequest("POST", "https://auth.globus.org/v2/oauth2/token/revoke", nil)
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	q := req.URL.Query()
	q.Set("token", token)
	if clientSecret == "" {
		q.Set("client_id", clientId)
	} else {
		req.Header.Add("Authorization", "Basic "+base64.StdEncoding.EncodeToString([]byte(clientId+":"+clientSecret)))
	}
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
