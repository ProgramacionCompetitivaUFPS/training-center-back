package group

import (
	"net/http"

	"github.com/training-judge-center/backend/internal/adapter/http/handler"
)

type deletionImpactResp struct {
	Contests       int `json:"contests"`
	ActiveContests int `json:"activeContests"`
	Materials      int `json:"materials"`
	Members        int `json:"members"`
	Submissions    int `json:"submissions"`
}

// @Summary      What deleting a group would take with it
// @Description  Counts of the contests (and how many are running now), materials, members and contest submissions that deleting the group removes or orphans. Same permission as deleting it: a lead of the group or an Admin.
// @Tags         groups
// @Produce      json
// @Security     BearerAuth
// @Param        groupId path string true "Group ID"
// @Success      200 {object} deletionImpactResp
// @Failure      401 {object} apperror.AppError
// @Failure      403 {object} apperror.AppError
// @Failure      404 {object} apperror.AppError
// @Router       /groups/{groupId}/deletion-impact [get]
func (h *Handler) GetDeletionImpact(w http.ResponseWriter, r *http.Request) {
	currentUser, ok := handler.RequireCurrentUser(w, r)
	if !ok {
		return
	}

	out, err := h.deleteGroup.Impact(r.Context(), r.PathValue("groupId"), *currentUser)
	if err != nil {
		handler.WriteError(r.Context(), w, err)
		return
	}

	handler.WriteJSON(r.Context(), w, http.StatusOK, deletionImpactResp{
		Contests:       out.Contests,
		ActiveContests: out.ActiveContests,
		Materials:      out.Materials,
		Members:        out.Members,
		Submissions:    out.Submissions,
	})
}

type soleLeadGroupResp struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type soleLeadGroupsResp struct {
	Groups []soleLeadGroupResp `json:"groups"`
}

// @Summary      Groups where I am the only lead
// @Description  Deactivating the account would leave these groups without a lead (only an Admin could manage them).
// @Tags         groups
// @Produce      json
// @Security     BearerAuth
// @Success      200 {object} soleLeadGroupsResp
// @Failure      401 {object} apperror.AppError
// @Router       /users/me/sole-lead-groups [get]
func (h *Handler) ListMySoleLeadGroups(w http.ResponseWriter, r *http.Request) {
	currentUser, ok := handler.RequireCurrentUser(w, r)
	if !ok {
		return
	}
	h.writeSoleLeadGroups(w, r, currentUser.ID)
}

// @Summary      Groups where a user is the only lead (Admin)
// @Tags         groups
// @Produce      json
// @Security     BearerAuth
// @Param        id path string true "User ID"
// @Success      200 {object} soleLeadGroupsResp
// @Failure      401 {object} apperror.AppError
// @Failure      403 {object} apperror.AppError
// @Router       /admin/users/{id}/sole-lead-groups [get]
func (h *Handler) ListUserSoleLeadGroups(w http.ResponseWriter, r *http.Request) {
	h.writeSoleLeadGroups(w, r, r.PathValue("id"))
}

func (h *Handler) writeSoleLeadGroups(w http.ResponseWriter, r *http.Request, userID string) {
	groups, err := h.listSoleLeadGroups.Execute(r.Context(), userID)
	if err != nil {
		handler.WriteError(r.Context(), w, err)
		return
	}
	out := soleLeadGroupsResp{Groups: make([]soleLeadGroupResp, 0, len(groups))}
	for _, g := range groups {
		out.Groups = append(out.Groups, soleLeadGroupResp{ID: g.ID, Name: g.Name})
	}
	handler.WriteJSON(r.Context(), w, http.StatusOK, out)
}
