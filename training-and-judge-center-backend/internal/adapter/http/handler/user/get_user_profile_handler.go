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

	out, err := h.getMyProfile.Execute(r.Context(), appuser.GetMyProfileInput{UserID: claims.UserID})
	if err != nil {
		handler.WriteError(w, err)
		return
	}

	handler.WriteJSON(w, http.StatusOK, buildFullResponse(out))
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

	out, err := h.getUserByNickname.Execute(r.Context(), appuser.GetUserByNicknameInput{
		RequesterID:   claims.UserID,
		RequesterRole: claims.Role,
		Nickname:      nickname,
	})
	if err != nil {
		handler.WriteError(w, err)
		return
	}

	if out.IsFullProfile {
		handler.WriteJSON(w, http.StatusOK, buildFullResponse(out))
	} else {
		handler.WriteJSON(w, http.StatusOK, publicUserResponse{
			Name:        out.User.Name,
			Nickname:    out.User.Nickname,
			Institution: out.User.Institution,
			Role:        out.User.Role,
			CreatedAt:   out.User.CreatedAt.Format("2006-01-02T15:04:05Z"),
		})
	}
}

func buildFullResponse(out *appuser.UserProfileOutput) fullUserResponse {
	resp := fullUserResponse{
		Name:        out.User.Name,
		Nickname:    out.User.Nickname,
		Country:     out.User.Country,
		City:        out.User.City,
		Institution: out.User.Institution,
		Role:        out.User.Role,
		CreatedAt:   out.User.CreatedAt.Format("2006-01-02T15:04:05Z"),
	}
	if out.User.Email != nil {
		resp.Email = *out.User.Email
	}
	if out.User.UpdatedAt != nil {
		resp.UpdatedAt = out.User.UpdatedAt.Format("2006-01-02T15:04:05Z")
	}
	return resp
}
