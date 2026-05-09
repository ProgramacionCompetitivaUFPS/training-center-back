package user

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/training-judge-center/backend/internal/adapter/http/handler"
	"github.com/training-judge-center/backend/internal/adapter/http/middleware"
	appuser "github.com/training-judge-center/backend/internal/application/user"
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

// @Summary      Get my profile
// @Tags         users
// @Produce      json
// @Security     BearerAuth
// @Success      200 {object} fullUserResponse
// @Failure      401 {object} apperror.AppError
// @Router       /users/me [get]
func (h *UserHandler) GetMyProfile(w http.ResponseWriter, r *http.Request) {
	claims := middleware.GetClaims(r.Context())
	if claims == nil {
		handler.WriteJSON(w, http.StatusUnauthorized, map[string]string{
			"error":   "UNAUTHORIZED",
			"message": "Invalid or missing authentication token",
		})
		return
	}

	result, err := h.getMyProfile.Execute(r.Context(), appuser.GetMyProfileInput{UserID: claims.UserID})
	if err != nil {
		handler.WriteError(w, err)
		return
	}

	handler.WriteJSON(w, http.StatusOK, buildFullResponse(result))
}

// @Summary      Get user by nickname
// @Tags         users
// @Produce      json
// @Security     BearerAuth
// @Param        nickname path string true "User nickname"
// @Success      200 {object} fullUserResponse
// @Failure      401 {object} apperror.AppError
// @Failure      404 {object} apperror.AppError
// @Router       /users/{nickname} [get]
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

	result, err := h.getUserByNickname.Execute(r.Context(), appuser.GetUserByNicknameInput{
		RequesterID:   claims.UserID,
		RequesterRole: claims.Role,
		Nickname:      nickname,
	})
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
