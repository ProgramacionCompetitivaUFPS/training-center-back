package material

import "github.com/training-judge-center/backend/pkg/apperror"

// NotMaterialAuthorError wraps a 403 AppError and carries the authorId so the
// delete handler can include it in the response body per spec.
type NotMaterialAuthorError struct {
	AuthorID string
	inner    *apperror.AppError
}

func NewNotMaterialAuthorError(authorID string) *NotMaterialAuthorError {
	return &NotMaterialAuthorError{
		AuthorID: authorID,
		inner:    apperror.NewForbidden(ErrCodeNotMaterialAuthor, "only the material author can delete this material"),
	}
}

func (e *NotMaterialAuthorError) Error() string { return e.inner.Error() }
func (e *NotMaterialAuthorError) Unwrap() error { return e.inner }
