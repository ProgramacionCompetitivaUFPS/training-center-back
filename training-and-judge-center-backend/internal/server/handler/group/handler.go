package group

import (
	"context"
	"net/http"

	appGroup "github.com/training-judge-center/backend/internal/application/group"
	"github.com/training-judge-center/backend/internal/domain/shared"
	"github.com/training-judge-center/backend/internal/server/handler"
	"github.com/training-judge-center/backend/internal/server/middleware"
	"github.com/training-judge-center/backend/pkg/apperror"
)

type listGroupsUC interface {
	Execute(ctx context.Context, in appGroup.ListGroupsInput) (*appGroup.ListGroupsOutput, error)
}

type getGroupUC interface {
	Execute(ctx context.Context, in appGroup.GetGroupInput) (*appGroup.GetGroupOutput, error)
}

type listMyGroupsUC interface {
	Execute(ctx context.Context, in appGroup.ListMyGroupsInput) (*appGroup.ListMyGroupsOutput, error)
}

type Handler struct {
	listUC   listGroupsUC
	getUC    getGroupUC
	listMyUC listMyGroupsUC
}

func NewHandler(listUC listGroupsUC, getUC getGroupUC, listMyUC listMyGroupsUC) *Handler {
	return &Handler{listUC: listUC, getUC: getUC, listMyUC: listMyUC}
}

func (h *Handler) requireCurrentUser(w http.ResponseWriter, r *http.Request) (*shared.CurrentUser, bool) {
	claims := middleware.GetClaims(r.Context())
	if claims == nil {
		handler.WriteJSON(w, http.StatusUnauthorized, apperror.AppError{
			Code:    apperror.ErrCodeUnauthorized,
			Message: "Invalid or missing authentication token",
		})
		return nil, false
	}
	u := shared.CurrentUser{ID: claims.UserID, Role: claims.Role.String()}
	return &u, true
}
