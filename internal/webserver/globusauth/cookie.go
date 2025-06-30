package globusauth

import (
	"fmt"
	"net/http"
	"time"

	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
)

func GetTokensFromCookie(ctx *gin.Context) (string, string, time.Time, error) {
	s := sessions.DefaultMany(ctx, "globus")
	refreshToken, ok1 := s.Get("refresh_token").(string)
	accessToken, ok2 := s.Get("access_token").(string)
	expiryStr, ok3 := s.Get("expiry").(string)

	if !(ok1 && ok2 && ok3) {
		return "", "", time.Time{}, fmt.Errorf("globus session has expired")
	}
	expiry, err := time.Parse(time.RFC3339Nano, expiryStr)
	if err != nil {
		return "", "", time.Time{}, fmt.Errorf("can't parse expiration time in cookie")
	}

	return refreshToken, accessToken, expiry, nil
}

func SetTokenCookie(ctx *gin.Context, refreshToken string, accessToken string, expiry time.Time, sessionDuration uint, secureCookies bool) error {
	s := sessions.DefaultMany(ctx, "globus")
	s.Set("refresh_token", refreshToken)
	s.Set("access_token", accessToken)
	s.Set("expiry", expiry.Format(time.RFC3339Nano))
	s.Options(sessions.Options{
		HttpOnly: true,
		MaxAge:   int(sessionDuration),
		Secure:   secureCookies || (ctx.Request.TLS != nil),
		SameSite: http.SameSiteNoneMode,
	})
	return s.Save()
}

func DeleteTokenCookie(ctx *gin.Context, secureCookies bool) error {
	s := sessions.DefaultMany(ctx, "globus")
	s.Delete("access_token")
	s.Delete("refresh_token")
	s.Delete("expiry")
	s.Options(sessions.Options{
		HttpOnly: true,
		Secure:   secureCookies || (ctx.Request.TLS != nil),
		SameSite: http.SameSiteNoneMode,
		MaxAge:   -1,
	})
	return s.Save()
}

func TestGlobusCookie(ctx *gin.Context) bool {
	globusSession := sessions.DefaultMany(ctx, "globus")
	rt, ok1 := globusSession.Get("refresh_token").(string)
	at, ok2 := globusSession.Get("access_token").(string)
	e, ok3 := globusSession.Get("expiry").(string)
	if !(ok1 && ok2 && ok3) || rt == "" || at == "" || e == "" {
		return false
	}
	_, err := time.Parse(time.RFC3339Nano, e)
	return err == nil
}
