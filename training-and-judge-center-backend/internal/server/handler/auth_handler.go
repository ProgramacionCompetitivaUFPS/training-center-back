package handler

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"

	"google.golang.org/api/idtoken"
)

type AuthHandler struct {
	clientID string
}

func NewAuthHandler(clientID string) *AuthHandler {
	return &AuthHandler{
		clientID: clientID,
	}
}

type GoogleLoginRequest struct {
	IDToken string `json:"idToken"`
}

type GoogleUser struct {
	Sub     string `json:"sub"`
	Email   string `json:"email"`
	Name    string `json:"name"`
	Picture string `json:"picture"`
}

func (h *AuthHandler) GoogleLogin(w http.ResponseWriter, r *http.Request) {
	var req GoogleLoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		slog.Error("invalid request body", "error", err)
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": "Invalid request body"})
		return
	}

	payload, err := idtoken.Validate(context.Background(), req.IDToken, h.clientID)
	if err != nil {
		slog.Error("invalid token", "error", err)
		respondJSON(w, http.StatusUnauthorized, map[string]string{"error": "Invalid token"})
		return
	}

	user := GoogleUser{
		Sub:     payload.Subject,
		Email:   payload.Claims["email"].(string),
		Name:    payload.Claims["name"].(string),
		Picture: payload.Claims["picture"].(string),
	}

	slog.Info("user authenticated", "email", user.Email)

	// TODO: Create or find user in database
	// TODO: Generate JWT token

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"user": user,
	})
}
