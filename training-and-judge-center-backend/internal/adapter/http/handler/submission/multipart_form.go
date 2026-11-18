package submission

import (
	"errors"
	"net/http"

	"github.com/training-judge-center/backend/pkg/apperror"
)

// maxSubmissionBodyBytes caps the whole multipart body at the parse budget, so a
// submission can never spill to a temporary file on disk.
const maxSubmissionBodyBytes = maxMultipartMemory

func parseSubmissionForm(w http.ResponseWriter, r *http.Request) *apperror.AppError {
	r.Body = http.MaxBytesReader(w, r.Body, maxSubmissionBodyBytes)
	if err := r.ParseMultipartForm(maxMultipartMemory); err != nil {
		var tooLarge *http.MaxBytesError
		if errors.As(err, &tooLarge) {
			return apperror.NewPayloadTooLarge("Request body is too large")
		}
		return apperror.NewBadRequest(apperror.ErrCodeBadRequest, "failed to parse multipart form")
	}
	return nil
}
