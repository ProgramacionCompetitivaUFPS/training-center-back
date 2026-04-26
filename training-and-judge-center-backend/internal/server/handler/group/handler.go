package group

import (
	"net/http"

	appGroup "github.com/training-judge-center/backend/internal/application/group"
	"github.com/training-judge-center/backend/internal/domain/shared"
	"github.com/training-judge-center/backend/internal/server/handler"
	"github.com/training-judge-center/backend/internal/server/middleware"
	"github.com/training-judge-center/backend/pkg/apperror"
)

type Handler struct {
	listUC   *appGroup.ListGroupsUseCase
	getUC    *appGroup.GetGroupUseCase
	listMyUC *appGroup.ListMyGroupsUseCase
}

func NewHandler(listUC *appGroup.ListGroupsUseCase, getUC *appGroup.GetGroupUseCase, listMyUC *appGroup.ListMyGroupsUseCase) *Handler {
	return &Handler{listUC: listUC, getUC: getUC, listMyUC: listMyUC}
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
