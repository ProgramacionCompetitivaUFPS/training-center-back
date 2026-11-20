package submission

import (
	"errors"
	"net/http"

	"github.com/training-judge-center/backend/pkg/apperror"
)

// maxSubmissionBodyBytes caps the whole multipart body: the source is limited to 1 MiB and the
// Blockly workspace to 512 KiB, so 2 MiB leaves room for framing and means a
// submission is never spilled to a temporary file on disk.
const maxSubmissionBodyBytes = 2 << 20

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
