package contest

import (
	"time"

	"github.com/training-judge-center/backend/internal/domain/shared"
	"github.com/training-judge-center/backend/pkg/apperror"
)

const MaxDescriptionLength = 5000

type Contest struct {
	id                    string
	name                  ContestName
	description           *string
	startTime             time.Time
	endTime               time.Time
	penalty               Penalty
	freezeMinutes         int
	enablePostContest     bool
	locked                bool
	showTeamMembers       bool
	allowLateRegistration bool
	resultsRevealedAt     *time.Time
	autoRevealMinutes     *int
	groupID               shared.GroupID
	ownerID               shared.UserID
	participationMode     ParticipationMode
	teamSize              TeamSize
	problems              []ContestProblem
	createdAt             time.Time
	updatedAt             *time.Time
	loadedUpdatedAt       *time.Time
}

func NewContest(
	id string,
	name ContestName,
	description *string,
	startTime, endTime time.Time,
	penalty Penalty,
	freezeMinutes int,
	enablePostContest bool,
	showTeamMembers bool,
	allowLateRegistration bool,
	groupID shared.GroupID,
	ownerID shared.UserID,
	participationMode ParticipationMode,
	teamSize TeamSize,
	now time.Time,
) (*Contest, error) {
	if id == "" {
		return nil, apperror.NewInternal()
	}
	if err := validateDescription(description); err != nil {
		return nil, err
	}
	t := now.UTC()
	if !startTime.After(t) {
		return nil, apperror.NewBadRequest(ErrCodeStartTimeInPast, "start time must be in the future")
	}
	if !endTime.After(startTime) {
		return nil, apperror.NewBadRequest(ErrCodeInvalidTimeRange, "end time must be after start time")
	}
	if err := validateFreezeMinutes(freezeMinutes, endTime.Sub(startTime)); err != nil {
		return nil, err
	}

	return &Contest{
		id:                    id,
		name:                  name,
		description:           description,
		startTime:             startTime.UTC(),
		endTime:               endTime.UTC(),
		penalty:               penalty,
		freezeMinutes:         freezeMinutes,
		enablePostContest:     enablePostContest,
		locked:                false,
		showTeamMembers:       showTeamMembers,
		allowLateRegistration: allowLateRegistration,
		groupID:               groupID,
		ownerID:               ownerID,
		participationMode:     participationMode,
		teamSize:              teamSize,
		problems:              []ContestProblem{},
		createdAt:             t,
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
	showTeamMembers bool,
	allowLateRegistration bool,
	resultsRevealedAt *time.Time,
	autoRevealMinutes *int,
	groupID shared.GroupID,
	ownerID shared.UserID,
	participationMode ParticipationMode,
	teamSize TeamSize,
	problems []ContestProblem,
	createdAt time.Time,
	updatedAt *time.Time,
) *Contest {
	if problems == nil {
		problems = []ContestProblem{}
	}
	var loadedUpdatedAt *time.Time
	if updatedAt != nil {
		t := *updatedAt
		loadedUpdatedAt = &t
	}
	return &Contest{
		id:                    id,
		name:                  name,
		description:           description,
		startTime:             startTime,
		endTime:               endTime,
		penalty:               penalty,
		freezeMinutes:         freezeMinutes,
		enablePostContest:     enablePostContest,
		locked:                locked,
		showTeamMembers:       showTeamMembers,
		allowLateRegistration: allowLateRegistration,
		resultsRevealedAt:     resultsRevealedAt,
		autoRevealMinutes:     autoRevealMinutes,
		groupID:               groupID,
		ownerID:               ownerID,
		participationMode:     participationMode,
		teamSize:              teamSize,
		problems:              problems,
		createdAt:             createdAt,
		updatedAt:             updatedAt,
		loadedUpdatedAt:       loadedUpdatedAt,
	}
}

// Status computes the contest status based on the provided current time.
func (c *Contest) Status(now time.Time) Status {
	if now.Before(c.startTime) {
		return StatusScheduled
	}
	if !now.Before(c.endTime) {
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

// Update applies optional field changes and validates time constraints.
// Pointer-nil means "do not change". description is **string so the caller can
// pass nil (no change) vs a pointer to nil (clear the description).
func (c *Contest) Update(
	name *ContestName,
	description **string,
	startTime, endTime *time.Time,
	penalty *Penalty,
	freezeMinutes *int,
	enablePostContest *bool,
	now time.Time,
) error {
	t := now.UTC()

	effStart := c.startTime
	effEnd := c.endTime
	if startTime != nil {
		if !startTime.UTC().After(t) {
			return apperror.NewBadRequest(ErrCodeStartTimeInPast, "start time must be in the future")
		}
		effStart = startTime.UTC()
	}

	if endTime != nil {
		if !endTime.UTC().After(t) {
			return apperror.NewBadRequest(ErrCodeEndTimeInPast, "end time must be in the future")
		}
		effEnd = endTime.UTC()
	}
	if (startTime != nil || endTime != nil) && !effEnd.After(effStart) {
		return apperror.NewBadRequest(ErrCodeInvalidTimeRange, "end time must be after start time")
	}

	if freezeMinutes != nil {
		if err := validateFreezeMinutes(*freezeMinutes, effEnd.Sub(effStart)); err != nil {
			return err
		}
	}
	if description != nil {
		if err := validateDescription(*description); err != nil {
			return err
		}
	}

	if name != nil {
		c.name = *name
	}
	if description != nil {
		c.description = *description
	}
	if startTime != nil {
		c.startTime = startTime.UTC()
	}
	if endTime != nil {
		c.endTime = endTime.UTC()
	}
	if penalty != nil {
		c.penalty = *penalty
	}
	if freezeMinutes != nil {
		c.freezeMinutes = *freezeMinutes
	}
	if enablePostContest != nil {
		c.enablePostContest = *enablePostContest
	}
	c.updatedAt = &t
	return nil
}

// SetParticipationMode updates the participation mode and touches updatedAt.
func (c *Contest) SetParticipationMode(mode ParticipationMode, now time.Time) {
	c.participationMode = mode
	t := now.UTC()
	c.updatedAt = &t
}

// SetTeamSize updates the team size and touches updatedAt.
func (c *Contest) SetTeamSize(size TeamSize, now time.Time) {
	c.teamSize = size
	t := now.UTC()
	c.updatedAt = &t
}

// SetLocked sets the locked flag and touches updatedAt.
func (c *Contest) SetLocked(locked bool, now time.Time) {
	c.locked = locked
	t := now.UTC()
	c.updatedAt = &t
}

// SetShowTeamMembers sets the show-team-members flag and touches updatedAt.
func (c *Contest) SetShowTeamMembers(showTeamMembers bool, now time.Time) {
	c.showTeamMembers = showTeamMembers
	t := now.UTC()
	c.updatedAt = &t
}

// SetAllowLateRegistration sets the allow-late-registration flag and touches updatedAt.
func (c *Contest) SetAllowLateRegistration(allow bool, now time.Time) {
	c.allowLateRegistration = allow
	t := now.UTC()
	c.updatedAt = &t
}

// FreezeTime returns the instant the scoreboard froze when it is frozen at now, nil otherwise.
// It stays frozen after the contest ends until the results are revealed (by hand, or automatically
// once AutoRevealAt passes); a contest without a freeze is never frozen.
func (c *Contest) FreezeTime(now time.Time) *time.Time {
	if c.freezeMinutes <= 0 {
		return nil
	}
	ft := c.endTime.Add(-time.Duration(c.freezeMinutes) * time.Minute)
	switch c.Status(now) {
	case StatusActive:
		if now.After(ft) {
			return &ft
		}
	case StatusFinished:
		if !c.resultsVisible(now) {
			return &ft
		}
	}
	return nil
}

// resultsVisible reports whether a finished contest's results are already open: revealed by hand or
// past their automatic reveal time.
func (c *Contest) resultsVisible(now time.Time) bool {
	if c.resultsRevealedAt != nil {
		return true
	}
	at := c.AutoRevealAt()
	return at != nil && !now.Before(*at)
}

// AutoRevealAt returns when the results open by themselves, nil when they only open by hand.
func (c *Contest) AutoRevealAt() *time.Time {
	if c.freezeMinutes <= 0 || c.autoRevealMinutes == nil {
		return nil
	}
	at := c.endTime.Add(time.Duration(*c.autoRevealMinutes) * time.Minute)
	return &at
}

// SetAutoRevealMinutes sets how long after the end the results open by themselves; nil means only by hand.
func (c *Contest) SetAutoRevealMinutes(minutes *int, now time.Time) error {
	if minutes != nil && *minutes < 0 {
		return apperror.NewValidation([]apperror.FieldError{
			{Field: "autoRevealMinutes", Message: "auto reveal minutes cannot be negative"},
		})
	}
	c.autoRevealMinutes = nil
	if minutes != nil {
		m := *minutes
		c.autoRevealMinutes = &m
	}
	t := now.UTC()
	c.updatedAt = &t
	return nil
}

// RevealResults unfreezes the scoreboard for everyone. Revealing twice keeps the first instant.
func (c *Contest) RevealResults(now time.Time) error {
	if c.freezeMinutes <= 0 {
		return apperror.NewConflict(ErrCodeContestHasNoFreeze, "the contest has no freeze, so there are no results to reveal")
	}
	if c.Status(now) != StatusFinished {
		return apperror.NewConflict(ErrCodeContestNotFinished, "results can only be revealed once the contest has finished")
	}
	if c.resultsVisible(now) {
		return nil
	}
	t := now.UTC()
	c.resultsRevealedAt = &t
	c.updatedAt = &t
	return nil
}

// CheckRegistrationOpen returns nil when new registrations are accepted at now.
func (c *Contest) CheckRegistrationOpen(now time.Time) error {
	switch c.Status(now) {
	case StatusScheduled:
		return nil
	case StatusActive:
		if c.allowLateRegistration {
			return nil
		}
		return apperror.NewConflict(ErrCodeContestAlreadyStarted, "registration is closed because the contest has already started")
	default:
		return apperror.NewConflict(ErrCodeContestNotOpenForRegistration, "the contest has finished and is not open for registration")
	}
}

// SetProblems replaces the entire problem list atomically.
func (c *Contest) SetProblems(problems []ContestProblem, now time.Time) {
	if problems == nil {
		problems = []ContestProblem{}
	}
	c.problems = problems
	t := now.UTC()
	c.updatedAt = &t
}

func (c *Contest) ID() string                           { return c.id }
func (c *Contest) Name() ContestName                    { return c.name }
func (c *Contest) Description() *string                 { return c.description }
func (c *Contest) StartTime() time.Time                 { return c.startTime }
func (c *Contest) EndTime() time.Time                   { return c.endTime }
func (c *Contest) Penalty() Penalty                     { return c.penalty }
func (c *Contest) FreezeMinutes() int                   { return c.freezeMinutes }
func (c *Contest) EnablePostContest() bool              { return c.enablePostContest }
func (c *Contest) Locked() bool                         { return c.locked }
func (c *Contest) ShowTeamMembers() bool                { return c.showTeamMembers }
func (c *Contest) AllowLateRegistration() bool          { return c.allowLateRegistration }
func (c *Contest) ResultsRevealedAt() *time.Time        { return c.resultsRevealedAt }
func (c *Contest) AutoRevealMinutes() *int              { return c.autoRevealMinutes }
func (c *Contest) GroupID() shared.GroupID              { return c.groupID }
func (c *Contest) OwnerID() shared.UserID               { return c.ownerID }
func (c *Contest) ParticipationMode() ParticipationMode { return c.participationMode }
func (c *Contest) TeamSize() TeamSize                   { return c.teamSize }
func (c *Contest) Problems() []ContestProblem           { return c.problems }
func (c *Contest) CreatedAt() time.Time                 { return c.createdAt }
func (c *Contest) UpdatedAt() *time.Time                { return c.updatedAt }

func (c *Contest) ConcurrencyToken() *time.Time {
	if c.loadedUpdatedAt == nil {
		return nil
	}
	t := *c.loadedUpdatedAt
	return &t
}

func validateDescription(d *string) error {
	if d == nil {
		return nil
	}
	if len([]rune(*d)) > MaxDescriptionLength {
		return apperror.NewValidation([]apperror.FieldError{
			{Field: "description", Message: "description cannot exceed 5000 characters"},
		})
	}
	return nil
}

func validateFreezeMinutes(v int, duration time.Duration) error {
	if v < 0 {
		return apperror.NewValidation([]apperror.FieldError{
			{Field: "freezeMinutes", Message: "freeze minutes cannot be negative"},
		})
	}
	if v > 0 && time.Duration(v)*time.Minute >= duration {
		return apperror.NewBadRequest(ErrCodeFreezeTooLong, "freeze time must be shorter than the contest duration")
	}
	return nil
}
