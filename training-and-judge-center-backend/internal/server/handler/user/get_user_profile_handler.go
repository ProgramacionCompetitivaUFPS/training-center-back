package user

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	appuser "github.com/training-judge-center/backend/internal/application/user"
	"github.com/training-judge-center/backend/internal/server/handler"
	"github.com/training-judge-center/backend/internal/server/middleware"
)

type fullUserResponse struct {
	Email       string `json:"email"`
	Name        string `json:"name"`
	Nickname    string `json:"nickname"`
	Country     string `json:"country"`
	City        string `json:"city"`
	Institution string `json:"institution"`
	Role        string `json:"role"`
	CreatedAt   string `json:"createdAt"`
	UpdatedAt   string `json:"updatedAt,omitempty"`
}

type publicUserResponse struct {
	Name        string `json:"name"`
	Nickname    string `json:"nickname"`
	Institution string `json:"institution"`
	Role        string `json:"role"`
	CreatedAt   string `json:"createdAt"`
}

func (h *UserHandler) GetMyProfile(w http.ResponseWriter, r *http.Request) {
	claims := middleware.GetClaims(r.Context())
	if claims == nil {
		handler.WriteJSON(w, http.StatusUnauthorized, map[string]string{
			"error":   "UNAUTHORIZED",
			"message": "Invalid or missing authentication token",
		})
		return
	}

	result, err := h.getUserProfile.GetMyProfile(r.Context(), claims.UserID)
	if err != nil {
		handler.WriteError(w, err)
		return
	}

	handler.WriteJSON(w, http.StatusOK, buildFullResponse(result))
}

func (h *UserHandler) GetByNickname(w http.ResponseWriter, r *http.Request) {
	claims := middleware.GetClaims(r.Context())
	if claims == nil {
		handler.WriteJSON(w, http.StatusUnauthorized, map[string]string{
			"error":   "UNAUTHORIZED",
			"message": "Invalid or missing authentication token",
		})
		return
	}

	nickname := chi.URLParam(r, "nickname")

	result, err := h.getUserProfile.GetUserByNickname(r.Context(), claims.UserID, claims.Role, nickname)
	if err != nil {
		handler.WriteError(w, err)
		return
	}

	if result.IsFullProfile {
		handler.WriteJSON(w, http.StatusOK, buildFullResponse(result))
	} else {
		handler.WriteJSON(w, http.StatusOK, publicUserResponse{
			Name:        result.User.Name,
			Nickname:    result.User.Nickname,
			Institution: result.User.Institution,
			Role:        result.User.Role,
			CreatedAt:   result.User.CreatedAt.Format("2006-01-02T15:04:05Z"),
		})
	}
}

func buildFullResponse(result *appuser.UserProfileOutput) fullUserResponse {
	resp := fullUserResponse{
		Name:        result.User.Name,
		Nickname:    result.User.Nickname,
		Country:     result.User.Country,
		City:        result.User.City,
		Institution: result.User.Institution,
		Role:        result.User.Role,
		CreatedAt:   result.User.CreatedAt.Format("2006-01-02T15:04:05Z"),
	}
	if result.User.Email != nil {
		resp.Email = *result.User.Email
	}
	if result.User.UpdatedAt != nil {
		resp.UpdatedAt = result.User.UpdatedAt.Format("2006-01-02T15:04:05Z")
	}
	return resp
}
