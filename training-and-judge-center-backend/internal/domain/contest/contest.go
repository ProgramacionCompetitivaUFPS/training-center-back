package contest

import (
	"time"

	"github.com/training-judge-center/backend/internal/domain/shared"
	"github.com/training-judge-center/backend/pkg/apperror"
)

const MaxDescriptionLength = 5000

type Contest struct {
	id                string
	name              ContestName
	description       *string
	startTime         time.Time
	endTime           time.Time
	penalty           Penalty
	freezeMinutes     int
	enablePostContest bool
	locked            bool
	groupID           shared.GroupID
	ownerID           shared.UserID
	problems          []ContestProblem
	createdAt         time.Time
	updatedAt         *time.Time
}

func NewContest(
	id string,
	name ContestName,
	description *string,
	startTime, endTime time.Time,
	penalty Penalty,
	freezeMinutes int,
	enablePostContest bool,
	groupID shared.GroupID,
	ownerID shared.UserID,
	now time.Time,
) (*Contest, error) {
	if id == "" {
		return nil, apperror.NewInternal()
	}
	if err := validateDescription(description); err != nil {
		return nil, err
	}
	if err := validateFreezeMinutes(freezeMinutes); err != nil {
		return nil, err
	}
	t := now.UTC()
	if !startTime.After(t) {
		return nil, apperror.NewBadRequest(ErrCodeStartTimeInPast, "start time must be in the future")
	}
	if !endTime.After(startTime) {
		return nil, apperror.NewBadRequest(ErrCodeInvalidTimeRange, "end time must be after start time")
	}

	return &Contest{
		id:                id,
		name:              name,
		description:       description,
		startTime:         startTime.UTC(),
		endTime:           endTime.UTC(),
		penalty:           penalty,
		freezeMinutes:     freezeMinutes,
		enablePostContest: enablePostContest,
		locked:            false,
		groupID:           groupID,
		ownerID:           ownerID,
		problems:          []ContestProblem{},
		createdAt:         t,
	}, nil
}

func RestoreContest(
	id string,
	name ContestName,
	description *string,
	startTime, endTime time.Time,
	penalty Penalty,
	freezeMinutes int,
	enablePostContest bool,
	locked bool,
	groupID shared.GroupID,
	ownerID shared.UserID,
	problems []ContestProblem,
	createdAt time.Time,
	updatedAt *time.Time,
) *Contest {
	if problems == nil {
		problems = []ContestProblem{}
	}
	return &Contest{
		id:                id,
		name:              name,
		description:       description,
		startTime:         startTime,
		endTime:           endTime,
		penalty:           penalty,
		freezeMinutes:     freezeMinutes,
		enablePostContest: enablePostContest,
		locked:            locked,
		groupID:           groupID,
		ownerID:           ownerID,
		problems:          problems,
		createdAt:         createdAt,
		updatedAt:         updatedAt,
	}
}

// Status computes the contest status based on the provided current time.
func (c *Contest) Status(now time.Time) Status {
	if now.Before(c.startTime) {
		return StatusScheduled
	}
	if now.After(c.endTime) {
		return StatusFinished
	}
	return StatusActive
}

// Duration returns the contest duration in minutes.
func (c *Contest) Duration() int {
	return int(c.endTime.Sub(c.startTime).Minutes())
}

// AddProblem appends a problem to the contest. Duplicate problemIDs are silently ignored.
// Returns apperror.NewInternal() if id or problemID is empty — those are programming errors.
func (c *Contest) AddProblem(id, problemID string, now time.Time) error {
	if id == "" || problemID == "" {
		return apperror.NewInternal()
	}
	for _, p := range c.problems {
		if p.problemID == problemID {
			return nil
		}
	}
	order := len(c.problems) + 1
	c.problems = append(c.problems, newContestProblem(id, problemID, order))
	t := now.UTC()
	c.updatedAt = &t
	return nil
}

// RemoveProblem removes a problem by problemID. Returns false if not found.
func (c *Contest) RemoveProblem(problemID string, now time.Time) bool {
	idx := -1
	for i, p := range c.problems {
		if p.problemID == problemID {
			idx = i
			break
		}
	}
	if idx == -1 {
		return false
	}
	c.problems = append(c.problems[:idx], c.problems[idx+1:]...)
	for i := range c.problems {
		c.problems[i].order = i + 1
	}
	t := now.UTC()
	c.updatedAt = &t
	return true
}

func (c *Contest) ID() string                 { return c.id }
func (c *Contest) Name() ContestName          { return c.name }
func (c *Contest) Description() *string       { return c.description }
func (c *Contest) StartTime() time.Time       { return c.startTime }
func (c *Contest) EndTime() time.Time         { return c.endTime }
func (c *Contest) Penalty() Penalty           { return c.penalty }
func (c *Contest) FreezeMinutes() int         { return c.freezeMinutes }
func (c *Contest) EnablePostContest() bool    { return c.enablePostContest }
func (c *Contest) Locked() bool               { return c.locked }
func (c *Contest) GroupID() shared.GroupID    { return c.groupID }
func (c *Contest) OwnerID() shared.UserID     { return c.ownerID }
func (c *Contest) Problems() []ContestProblem { return c.problems }
func (c *Contest) CreatedAt() time.Time       { return c.createdAt }
func (c *Contest) UpdatedAt() *time.Time      { return c.updatedAt }

func validateDescription(d *string) error {
	if d == nil {
		return nil
	}
	if len([]rune(*d)) > MaxDescriptionLength {
		return apperror.NewBadRequest(ErrCodeDescriptionTooLong, "description cannot exceed 5000 characters")
	}
	return nil
}

func validateFreezeMinutes(v int) error {
	if v < 0 {
		return apperror.NewBadRequest(ErrCodeInvalidFreezeMinutes, "freeze minutes cannot be negative")
	}
	return nil
}
