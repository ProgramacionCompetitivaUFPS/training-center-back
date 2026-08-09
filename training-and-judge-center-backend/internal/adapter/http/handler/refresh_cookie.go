package handler

import (
	"net/http"
	"time"
)

const refreshCookieName = "refresh_token"
const refreshCookiePath = "/auth"

func setRefreshCookie(w http.ResponseWriter, secret string, expiresAt time.Time) {
	http.SetCookie(w, &http.Cookie{
		Name:     refreshCookieName,
		Value:    secret,
		Path:     refreshCookiePath,
		Expires:  expiresAt,
		MaxAge:   int(time.Until(expiresAt).Seconds()),
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteStrictMode,
	})
}

func readRefreshCookie(r *http.Request) (string, bool) {
	c, err := r.Cookie(refreshCookieName)
	if err != nil || c.Value == "" {
		return "", false
	}
	return c.Value, true
}
