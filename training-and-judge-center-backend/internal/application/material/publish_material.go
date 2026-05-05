package material

import (
	"context"
	"log/slog"
	"time"

	domainMaterial "github.com/training-judge-center/backend/internal/domain/material"
	"github.com/training-judge-center/backend/internal/domain/shared"
	appshared "github.com/training-judge-center/backend/internal/application/shared"
	"github.com/training-judge-center/backend/pkg/apperror"
)

type PublishMaterialInput struct {
	CurrentUser appshared.CurrentUser
	GroupID     string
	MaterialID  string
}

type PublishMaterialOutput struct {
	Material MaterialData
}

type PublishMaterial struct {
	repo           domainMaterial.Repository
	groupProvider  GroupProvider
	authorProvider AuthorProvider
}

func NewPublishMaterial(
	repo domainMaterial.Repository,
	groupProvider GroupProvider,
	authorProvider AuthorProvider,
) *PublishMaterial {
	return &PublishMaterial{repo: repo, groupProvider: groupProvider, authorProvider: authorProvider}
}

func (uc *PublishMaterial) Execute(ctx context.Context, in PublishMaterialInput) (*PublishMaterialOutput, error) {
	exists, err := uc.groupProvider.Exists(ctx, in.GroupID)
	if err != nil {
		slog.ErrorContext(ctx, "failed to check group existence", "error", err, "group_id", in.GroupID)
		return nil, apperror.NewInternal()
	}
	if !exists {
		return nil, apperror.NewNotFound(ErrCodeGroupNotFound, "group not found")
	}

	m, err := uc.repo.FindByID(ctx, in.MaterialID)
	if err != nil {
		return nil, err
	}
	if m.GroupID() != in.GroupID {
		return nil, apperror.NewNotFound(domainMaterial.ErrCodeMaterialNotFound, "material not found")
	}

	// Publish/unpublish is restricted to the author (and admins) — group leads are intentionally
	// excluded. Leads can pin/unpin to control visibility within the feed, but deciding whether
	// content is published at all is an authorship decision.
	if !m.CanBeEditedBy(shared.RestoreUserID(in.CurrentUser.ID), in.CurrentUser.IsAdmin()) {
		return nil, apperror.NewForbidden(ErrCodeNotMaterialAuthor, "only the material author can publish this material")
	}

	// Idempotent: already-published materials return 200 with current state.
	if !m.Status().IsPublished() {
		if err := m.Publish(time.Now()); err != nil {
			return nil, err
		}
		if err := uc.repo.Save(ctx, m); err != nil {
			slog.ErrorContext(ctx, "failed to save published material", "error", err, "material_id", in.MaterialID)
			return nil, apperror.NewInternal()
		}
	}

	data := toMaterialData(m)
	displays, err := uc.authorProvider.GetDisplays(ctx, []string{m.AuthorID().Value()})
	if err != nil {
		slog.WarnContext(ctx, "failed to resolve author display, returning without author info", "error", err, "author_id", m.AuthorID().Value())
	} else {
		data.Author = displays[m.AuthorID().Value()]
	}

	return &PublishMaterialOutput{Material: data}, nil
}
