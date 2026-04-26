package group

import (
	"time"

	"github.com/training-judge-center/backend/internal/domain/shared"
	"github.com/training-judge-center/backend/pkg/apperror"
)

type Group struct {
	id          string
	name        GroupName
	description *string
	visibility  Visibility
	joinPolicy  JoinPolicy
	isDefault   bool
	createdBy   shared.UserID
	createdAt   time.Time
	updatedAt   time.Time
	clock       func() time.Time
}

func (g *Group) now() time.Time { return g.clock().UTC() }

func (g *Group) WithClock(fn func() time.Time) *Group {
	g.clock = fn
	return g
}

// NewGroup constructs a validated Group. Pass a non-nil clock for deterministic
// timestamps in tests; nil defaults to time.Now.
func NewGroup(id string, name GroupName, description *string, visibility Visibility, joinPolicy JoinPolicy, createdBy shared.UserID, clock func() time.Time) (*Group, error) {
	if id == "" {
		return nil, apperror.NewInternal()
	}
	if err := validatePolicyCombination(visibility, joinPolicy); err != nil {
		return nil, err
	}
	if clock == nil {
		clock = time.Now
	}
	now := clock().UTC()
	return &Group{
		id:          id,
		name:        name,
		description: description,
		visibility:  visibility,
		joinPolicy:  joinPolicy,
		isDefault:   false,
		createdBy:   createdBy,
		createdAt:   now,
		updatedAt:   now,
		clock:       clock,
	}, nil
}

func RestoreGroup(id string, name GroupName, description *string, visibility Visibility, joinPolicy JoinPolicy, isDefault bool, createdBy shared.UserID, createdAt, updatedAt time.Time) *Group {
	return &Group{
		id:          id,
		name:        name,
		description: description,
		visibility:  visibility,
		joinPolicy:  joinPolicy,
		isDefault:   isDefault,
		createdBy:   createdBy,
		createdAt:   createdAt,
		updatedAt:   updatedAt,
		clock:       time.Now,
	}
}

func (g *Group) ID() string               { return g.id }
func (g *Group) Name() GroupName          { return g.name }
func (g *Group) Description() *string     { return g.description }
func (g *Group) Visibility() Visibility   { return g.visibility }
func (g *Group) JoinPolicy() JoinPolicy   { return g.joinPolicy }
func (g *Group) IsDefault() bool          { return g.isDefault }
func (g *Group) CreatedBy() shared.UserID { return g.createdBy }
func (g *Group) CreatedAt() time.Time     { return g.createdAt }
func (g *Group) UpdatedAt() time.Time     { return g.updatedAt }

func (g *Group) CanBeDeleted() bool { return !g.isDefault }

// UpdateMetadata updates name and/or description.
//   - Pass nil for name to leave it unchanged.
//   - Pass nil for description to leave it unchanged.
//   - Pass &nilPtr (where nilPtr is a nil *string) to clear the description.
//   - Pass &ptr (where ptr points to a string) to set a new value.
func (g *Group) UpdateMetadata(name *GroupName, description **string) {
	if name == nil && description == nil {
		return
	}
	if name != nil {
		g.name = *name
	}
	if description != nil {
		g.description = *description
	}
	g.updatedAt = g.now()
}

func (g *Group) UpdatePolicies(visibility *Visibility, joinPolicy *JoinPolicy) error {
	if visibility == nil && joinPolicy == nil {
		return nil
	}
	newV := g.visibility
	newJP := g.joinPolicy
	if visibility != nil {
		newV = *visibility
	}
	if joinPolicy != nil {
		newJP = *joinPolicy
	}
	if err := validatePolicyCombination(newV, newJP); err != nil {
		return err
	}
	g.visibility = newV
	g.joinPolicy = newJP
	g.updatedAt = g.now()
	return nil
}

func validatePolicyCombination(v Visibility, jp JoinPolicy) error {
	if v == VisibilityNotVisible && jp != JoinPolicyInvite {
		return apperror.NewValidation([]apperror.FieldError{
			{Field: "joinPolicy", Message: "non-visible groups can only use the INVITE join policy"},
		})
	}
	return nil
}
