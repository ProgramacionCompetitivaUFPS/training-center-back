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
func (h *Handler) GetMyProfile(w http.ResponseWriter, r *http.Request) {
	cu, ok := middleware.GetCurrentUser(r.Context())
	if !ok {
		handler.WriteJSON(r.Context(), w, http.StatusUnauthorized, map[string]string{
			"error":   "UNAUTHORIZED",
			"message": "Invalid or missing authentication token",
		})
		return
	}

	out, err := h.getMyProfile.Execute(r.Context(), appuser.GetMyProfileInput{UserID: cu.ID})
	if err != nil {
		handler.WriteError(r.Context(), w, err)
		return
	}

	handler.WriteJSON(r.Context(), w, http.StatusOK, buildFullResponse(out.User))
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
func (h *Handler) GetByNickname(w http.ResponseWriter, r *http.Request) {
	cu, ok := middleware.GetCurrentUser(r.Context())
	if !ok {
		handler.WriteJSON(r.Context(), w, http.StatusUnauthorized, map[string]string{
			"error":   "UNAUTHORIZED",
			"message": "Invalid or missing authentication token",
		})
		return
	}

	nickname := chi.URLParam(r, "nickname")

	out, err := h.getUserByNickname.Execute(r.Context(), appuser.GetUserByNicknameInput{
		RequesterID:   cu.ID,
		RequesterRole: cu.Role,
		Nickname:      nickname,
	})
	if err != nil {
		handler.WriteError(r.Context(), w, err)
		return
	}

	if out.IsFullProfile {
		handler.WriteJSON(r.Context(), w, http.StatusOK, buildFullResponse(out.User))
	} else {
		handler.WriteJSON(r.Context(), w, http.StatusOK, publicUserResponse{
			Name:        out.User.Name,
			Nickname:    out.User.Nickname,
			Institution: out.User.Institution,
			Role:        out.User.Role,
			CreatedAt:   out.User.CreatedAt.Format("2006-01-02T15:04:05Z"),
		})
	}
}

func buildFullResponse(u appuser.UserDTO) fullUserResponse {
	resp := fullUserResponse{
		Name:        u.Name,
		Nickname:    u.Nickname,
		Country:     u.Country,
		City:        u.City,
		Institution: u.Institution,
		Role:        u.Role,
		CreatedAt:   u.CreatedAt.Format("2006-01-02T15:04:05Z"),
	}
	if u.Email != nil {
		resp.Email = *u.Email
	}
	if u.UpdatedAt != nil {
		resp.UpdatedAt = u.UpdatedAt.Format("2006-01-02T15:04:05Z")
	}
	return resp
}
