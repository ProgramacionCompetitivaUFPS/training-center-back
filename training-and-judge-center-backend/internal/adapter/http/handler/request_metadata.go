package handler

import (
	"net"
	"net/http"
)

func clientIP(r *http.Request) *string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil || host == "" {
		return nil
	}
	return &host
}

func userAgent(r *http.Request) *string {
	ua := r.Header.Get("User-Agent")
	if ua == "" {
		return nil
	}
	return &ua
}
