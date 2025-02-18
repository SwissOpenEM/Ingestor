package refreshfunctoken

import (
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/oauth2"
)

type TokenSource struct {
	ctx             *gin.Context
	oc              *oauth2.Config
	accessToken     string
	refreshToken    string
	expiry          time.Time
	sessionDuration uint
	refreshFunc     func(ctx *gin.Context, globusConf *oauth2.Config, refreshToken string, sessionDuration uint) (string, string, time.Time, error)
	tokenMutex      sync.Mutex
}

func (ts *TokenSource) Token() (*oauth2.Token, error) {
	ts.tokenMutex.Lock()
	defer ts.tokenMutex.Unlock()

	if time.Now().After(ts.expiry) || ts.accessToken == "" {
		accessToken, refreshToken, expiry, err := ts.refreshFunc(ts.ctx, ts.oc, ts.refreshToken, ts.sessionDuration)
		if err != nil {
			return nil, err
		}
		ts.accessToken = accessToken
		ts.refreshToken = refreshToken
		ts.expiry = expiry
	}

	return &oauth2.Token{
		AccessToken:  ts.accessToken,
		RefreshToken: ts.refreshToken,
		Expiry:       ts.expiry,
		TokenType:    "Bearer",
	}, nil
}

func NewTokenSource(
	ctx *gin.Context,
	oc *oauth2.Config,
	accessToken string,
	refreshToken string,
	expiry time.Time,
	sessionDuration uint,
	refreshFunc func(ctx *gin.Context, oc *oauth2.Config, refreshToken string, sessionDuration uint) (string, string, time.Time, error),
) *TokenSource {
	return &TokenSource{
		ctx:             ctx,
		oc:              oc,
		accessToken:     accessToken,
		refreshToken:    refreshToken,
		expiry:          expiry,
		sessionDuration: sessionDuration,
		refreshFunc:     refreshFunc,
	}
}
