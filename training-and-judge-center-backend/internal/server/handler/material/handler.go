package material

import (
	"net/http"

	appMaterial "github.com/training-judge-center/backend/internal/application/material"
	"github.com/training-judge-center/backend/internal/domain/shared"
	"github.com/training-judge-center/backend/internal/server/handler"
	"github.com/training-judge-center/backend/internal/server/middleware"
	"github.com/training-judge-center/backend/pkg/apperror"
)

type Handler struct {
	createUC *appMaterial.CreateMaterial
	updateUC *appMaterial.UpdateMaterial
}

func NewHandler(createUC *appMaterial.CreateMaterial, updateUC *appMaterial.UpdateMaterial) *Handler {
	return &Handler{createUC: createUC, updateUC: updateUC}
}

func (h *Handler) requireCurrentUser(w http.ResponseWriter, r *http.Request) (*shared.CurrentUser, bool) {
	claims := middleware.GetClaims(r.Context())
	if claims == nil {
		handler.WriteError(w, apperror.NewUnauthorized(apperror.ErrCodeUnauthorized, "Invalid or missing authentication token"))
		return nil, false
	}
	u := shared.CurrentUser{ID: claims.UserID, Role: claims.Role.String()}
	return &u, true
}
