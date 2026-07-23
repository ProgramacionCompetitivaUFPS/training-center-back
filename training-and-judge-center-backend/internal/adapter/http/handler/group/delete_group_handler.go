package group

import (
	"encoding/json"
	"io"
	"net/http"

	"github.com/training-judge-center/backend/internal/adapter/http/handler"
	appGroup "github.com/training-judge-center/backend/internal/application/group"
	"github.com/training-judge-center/backend/pkg/apperror"
)

// @Summary      Delete group
// @Tags         groups
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        groupId path string         true "Group ID"
// @Param        body    body deleteGroupReq true "Confirmation name"
// @Success      200 {object} deleteGroupResp
// @Failure      400 {object} apperror.AppError
// @Failure      401 {object} apperror.AppError
// @Failure      403 {object} apperror.AppError
// @Failure      404 {object} apperror.AppError
// @Router       /groups/{groupId} [delete]
func (h *Handler) DeleteGroup(w http.ResponseWriter, r *http.Request) {
	currentUser, ok := handler.RequireCurrentUser(w, r)
	if !ok {
		return
	}

	var req deleteGroupReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil && err != io.EOF {
		handler.WriteError(r.Context(), w, apperror.NewValidation([]apperror.FieldError{
			{Field: "body", Message: "invalid JSON"},
		}))
		return
	}

	out, err := h.deleteGroup.Execute(r.Context(), appGroup.DeleteGroupInput{
		GroupID:          r.PathValue("groupId"),
		ConfirmationName: req.ConfirmationName,
		CurrentUser:      *currentUser,
	})
	if err != nil {
		handler.WriteError(r.Context(), w, err)
		return
	}

	handler.WriteJSON(r.Context(), w, http.StatusOK, deleteGroupResp{
		Message: "Group deleted successfully",
		DeletedGroup: deletedGroupInfo{
			ID:   out.GroupID,
			Name: out.GroupName,
		},
		DeletionSummary: deletionSummary{
			ContestsDeleted:            out.ContestsDeleted,
			MaterialsDeleted:           out.MaterialsDeleted,
			StandingCollectionsDeleted: out.StandingsDeleted,
			SubmissionsOrphaned:        out.SubmissionsOrphaned,
			MembersRemoved:             out.MembersRemoved,
		},
	})
}
