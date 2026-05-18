package contest

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/training-judge-center/backend/internal/adapter/http/handler"
	appContest "github.com/training-judge-center/backend/internal/application/contest"
	"github.com/training-judge-center/backend/pkg/apperror"
)

func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	caller, ok := h.requireCurrentUser(w, r)
	if !ok {
		return
	}

	groupID := chi.URLParam(r, "groupId")
	if groupID == "" {
		handler.WriteError(r.Context(), w, apperror.NewBadRequest(apperror.ErrCodeBadRequest, "missing groupId"))
		return
	}

	var body createContestRequest
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		handler.WriteError(r.Context(), w, apperror.NewValidation([]apperror.FieldError{
			{Field: "body", Message: "invalid JSON"},
		}))
		return
	}

	out, err := h.createContest.Execute(r.Context(), appContest.CreateContestInput{
		CurrentUser:       *caller,
		GroupID:           groupID,
		Name:              body.Name,
		Description:       body.Description,
		StartTime:         body.StartTime,
		EndTime:           body.EndTime,
		Penalty:           body.Penalty,
		FreezeMinutes:     body.FreezeMinutes,
		EnablePostContest: body.EnablePostContest,
		Problems:          body.Problems,
	})
	if err != nil {
		handler.WriteError(r.Context(), w, err)
		return
	}

	handler.WriteJSON(r.Context(), w, http.StatusCreated, toContestResponse(out))
}
