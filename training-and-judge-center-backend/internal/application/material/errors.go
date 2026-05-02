package material

import "github.com/training-judge-center/backend/pkg/apperror"

const (
	ErrCodeGroupNotFound     = "GROUP_NOT_FOUND"
	ErrCodeInsufficientPerms = "INSUFFICIENT_PERMISSIONS"
	ErrCodeNotMaterialAuthor = "NOT_MATERIAL_AUTHOR"
)

// NotMaterialAuthorError is used by DeleteMaterial to carry the authorId in the 403 response.
type NotMaterialAuthorError struct {
	AuthorID string
	inner    *apperror.AppError
}

func newNotMaterialAuthorError(authorID string) *NotMaterialAuthorError {
	return &NotMaterialAuthorError{
		AuthorID: authorID,
		inner:    apperror.NewForbidden(ErrCodeNotMaterialAuthor, "only the material author can delete this material"),
	}
}

func (e *NotMaterialAuthorError) Error() string { return e.inner.Error() }
func (e *NotMaterialAuthorError) Unwrap() error { return e.inner }
