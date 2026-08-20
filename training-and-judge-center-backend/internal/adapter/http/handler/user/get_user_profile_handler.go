package user

import (
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/training-judge-center/backend/internal/adapter/http/handler"
	"github.com/training-judge-center/backend/internal/adapter/http/middleware"
	appuser "github.com/training-judge-center/backend/internal/application/user"
	"github.com/training-judge-center/backend/pkg/apperror"
)

type fullUserResponse struct {
	Email        string `json:"email"`
	Name         string `json:"name"`
	Nickname     string `json:"nickname"`
	Country      string `json:"country"`
	City         string `json:"city"`
	Institution  string `json:"institution"`
	Role         string `json:"role"`
	CreatedAt    string `json:"createdAt"`
	UpdatedAt    string `json:"updatedAt,omitempty"`
	GoogleLinked bool   `json:"googleLinked"`
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
		handler.WriteJSON(r.Context(), w, http.StatusUnauthorized, apperror.AppError{Code: apperror.ErrCodeUnauthorized, Message: "invalid or missing authentication token"})
		return
	}

	out, err := h.getMyProfile.Execute(r.Context(), appuser.GetMyProfileInput{UserID: cu.ID})
	if err != nil {
		handler.WriteError(r.Context(), w, err)
		return
	}

	resp := fullUserResponse{
		Name:         out.User.Name,
		Nickname:     out.User.Nickname,
		Country:      out.User.Country,
		City:         out.User.City,
		Institution:  out.User.Institution,
		Role:         out.User.Role,
		CreatedAt:    out.User.CreatedAt.UTC().Format(time.RFC3339),
		GoogleLinked: out.GoogleLinked,
	}
	if out.User.Email != nil {
		resp.Email = *out.User.Email
	}
	if out.User.UpdatedAt != nil {
		resp.UpdatedAt = out.User.UpdatedAt.UTC().Format(time.RFC3339)
	}

	handler.WriteJSON(r.Context(), w, http.StatusOK, resp)
}

// @Summary      Get user by nickname
// @Tags         users
// @Produce      json
// @Security     BearerAuth
// @Param        nickname path string true "User nickname"
// @Success      200 {object} updatedUserResponse
// @Failure      401 {object} apperror.AppError
// @Failure      404 {object} apperror.AppError
// @Router       /users/{nickname} [get]
func (h *Handler) GetByNickname(w http.ResponseWriter, r *http.Request) {
	cu, ok := middleware.GetCurrentUser(r.Context())
	if !ok {
		handler.WriteJSON(r.Context(), w, http.StatusUnauthorized, apperror.AppError{Code: apperror.ErrCodeUnauthorized, Message: "invalid or missing authentication token"})
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
		handler.WriteJSON(r.Context(), w, http.StatusOK, buildUpdatedResponse(out.User))
	} else {
		handler.WriteJSON(r.Context(), w, http.StatusOK, publicUserResponse{
			Name:        out.User.Name,
			Nickname:    out.User.Nickname,
			Institution: out.User.Institution,
			Role:        out.User.Role,
			CreatedAt:   out.User.CreatedAt.UTC().Format(time.RFC3339),
		})
	}
}
