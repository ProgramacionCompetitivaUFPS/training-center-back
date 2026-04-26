package group

import (
	"context"
	"errors"
	"log/slog"

	"github.com/google/uuid"
	domainGroup "github.com/training-judge-center/backend/internal/domain/group"
	"github.com/training-judge-center/backend/internal/domain/shared"
	"github.com/training-judge-center/backend/pkg/apperror"
)

type CreateGroupInput struct {
	Name        string
	Description *string
	JoinMode    string
	Visibility  string
	CurrentUser shared.CurrentUser
}

type CreateGroupResult struct {
	Group *domainGroup.Group
}

type CreateGroupUseCase struct {
	repo domainGroup.Repository
}

func NewCreateGroupUseCase(repo domainGroup.Repository) *CreateGroupUseCase {
	return &CreateGroupUseCase{repo: repo}
}

func (uc *CreateGroupUseCase) Execute(ctx context.Context, input CreateGroupInput) (*CreateGroupResult, error) {
	if input.CurrentUser.Role != shared.RoleCoach && !input.CurrentUser.IsAdmin() {
		return nil, apperror.NewForbidden(
			domainGroup.ErrCodeInsufficientPermissions,
			"Only Admin and Coach users can create groups",
		)
	}

	var fieldErrs []apperror.FieldError

	groupName, err := domainGroup.NewGroupName(input.Name)
	if err := appendFieldErr(err, "name", &fieldErrs); err != nil {
		return nil, err
	}

	joinPolicy, err := domainGroup.NewJoinPolicy(input.JoinMode)
	if err := appendFieldErr(err, "joinMode", &fieldErrs); err != nil {
		return nil, err
	}

	visibility, err := domainGroup.NewVisibility(input.Visibility)
	if err := appendFieldErr(err, "visibility", &fieldErrs); err != nil {
		return nil, err
	}

	if len(fieldErrs) > 0 {
		return nil, apperror.NewValidation(fieldErrs)
	}

	exists, err := uc.repo.ExistsByName(ctx, groupName)
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, apperror.NewConflict(domainGroup.ErrCodeNameAlreadyExists, "A group with this name already exists")
	}

	newID := uuid.New().String()
	g, err := domainGroup.NewGroup(
		newID,
		groupName,
		input.Description,
		visibility,
		joinPolicy,
		shared.RestoreUserID(input.CurrentUser.ID),
		nil,
	)
	if err != nil {
		// NewGroup retorna ErrCodeInvalidPolicyCombination si visibility+joinPolicy es inválida.
		return nil, err
	}

	if err := uc.repo.Save(ctx, g); err != nil {
		slog.ErrorContext(ctx, "failed to save new group", "error", err, "group_id", newID)
		return nil, apperror.NewInternal()
	}

	return &CreateGroupResult{Group: g}, nil
}

// appendFieldErr convierte un AppError de dominio (sin Details) en un FieldError acumulable.
// Los validadores del dominio group retornan NewBadRequest sin Details, por lo que
// AccumulateFieldErrors los consumiría silenciosamente. Este helper extrae el mensaje
// y lo empaqueta con el nombre de campo correcto para la respuesta al cliente.
func appendFieldErr(err error, field string, fieldErrs *[]apperror.FieldError) error {
	if err == nil {
		return nil
	}
	var ae *apperror.AppError
	if errors.As(err, &ae) {
		*fieldErrs = append(*fieldErrs, apperror.FieldError{Field: field, Message: ae.Message})
		return nil
	}
	return apperror.NewInternal()
}
