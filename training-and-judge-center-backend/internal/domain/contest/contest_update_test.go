package contest_test

import (
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/training-judge-center/backend/internal/domain/contest"
	"github.com/training-judge-center/backend/internal/domain/shared"
)

type updateArgs struct {
	name        *contest.ContestName
	description **string
	start, end  *time.Time
	penalty     *contest.Penalty
	freeze      *int
	post        *bool
}

func (a updateArgs) apply(c *contest.Contest, now time.Time) error {
	return c.Update(a.name, a.description, a.start, a.end, a.penalty, a.freeze, a.post, now)
}

type contestFields struct {
	Name        string
	Description *string
	Start, End  time.Time
	Penalty     int
	Freeze      int
	Post        bool
	UpdatedAt   *time.Time
}

func fieldsOf(c *contest.Contest) contestFields {
	return contestFields{
		Name:        c.Name().Value(),
		Description: c.Description(),
		Start:       c.StartTime(),
		End:         c.EndTime(),
		Penalty:     c.Penalty().Value(),
		Freeze:      c.FreezeMinutes(),
		Post:        c.EnablePostContest(),
		UpdatedAt:   c.UpdatedAt(),
	}
}

func ptr[T any](v T) *T { return &v }

func contestWithDescription(t *testing.T, description *string) *contest.Contest {
	t.Helper()
	c, err := contest.NewContest(
		"contest-id",
		mustName(t, "Weekly Contest"),
		description,
		fixedNow.Add(time.Hour),
		fixedNow.Add(4*time.Hour),
		mustPenalty(t, 20),
		60,
		false,
		false, false,
		shared.RestoreGroupID("group-id"),
		shared.RestoreUserID("owner-id"),
		contest.RestoreParticipationMode("INDIVIDUAL"), contest.RestoreTeamSize(2, 5),
		fixedNow,
	)
	if err != nil {
		t.Fatalf("contestWithDescription: %v", err)
	}
	return c
}

func TestUpdate_NoArguments_ChangesNothingButTheUpdateInstant(t *testing.T) {
	c := validContest(t)
	before := fieldsOf(c)
	now := time.Date(2026, 5, 4, 12, 30, 0, 0, time.UTC)

	err := updateArgs{}.apply(c, now)

	if err != nil {
		t.Fatalf("Update: %v", err)
	}
	after := fieldsOf(c)
	if c.UpdatedAt() == nil || !c.UpdatedAt().Equal(now) {
		t.Errorf("UpdatedAt = %v, want %v", c.UpdatedAt(), now)
	}
	before.UpdatedAt, after.UpdatedAt = nil, nil
	if !reflect.DeepEqual(before, after) {
		t.Errorf("fields changed without arguments: before %+v, after %+v", before, after)
	}
}

func TestUpdate_AppliesEveryProvidedField(t *testing.T) {
	c := validContest(t)
	newDescription := ptr("New text")
	start := time.Date(2026, 5, 4, 14, 0, 0, 0, time.UTC)
	end := time.Date(2026, 5, 4, 18, 0, 0, 0, time.UTC)

	err := updateArgs{
		name:        ptr(mustName(t, "Renamed")),
		description: &newDescription,
		start:       &start,
		end:         &end,
		penalty:     ptr(mustPenalty(t, 45)),
		freeze:      ptr(30),
		post:        ptr(true),
	}.apply(c, fixedNow)

	if err != nil {
		t.Fatalf("Update: %v", err)
	}
	if c.Name().Value() != "Renamed" {
		t.Errorf("Name = %q, want Renamed", c.Name().Value())
	}
	if c.Description() == nil || *c.Description() != "New text" {
		t.Errorf("Description = %v, want New text", c.Description())
	}
	if !c.StartTime().Equal(start) || !c.EndTime().Equal(end) {
		t.Errorf("times = %v..%v, want %v..%v", c.StartTime(), c.EndTime(), start, end)
	}
	if c.Penalty().Value() != 45 {
		t.Errorf("Penalty = %d, want 45", c.Penalty().Value())
	}
	if c.FreezeMinutes() != 30 {
		t.Errorf("FreezeMinutes = %d, want 30", c.FreezeMinutes())
	}
	if !c.EnablePostContest() {
		t.Error("EnablePostContest = false, want true")
	}
}

func TestUpdate_Times_AreStoredInUTC(t *testing.T) {
	c := validContest(t)
	plusTwo := time.FixedZone("UTC+2", 2*3600)
	start := time.Date(2026, 5, 4, 16, 0, 0, 0, plusTwo)
	end := time.Date(2026, 5, 4, 20, 0, 0, 0, plusTwo)

	err := updateArgs{start: &start, end: &end}.apply(c, fixedNow)

	if err != nil {
		t.Fatalf("Update: %v", err)
	}
	if c.StartTime().Location() != time.UTC || c.EndTime().Location() != time.UTC {
		t.Errorf("times are not in UTC: %v, %v", c.StartTime().Location(), c.EndTime().Location())
	}
	if want := time.Date(2026, 5, 4, 14, 0, 0, 0, time.UTC); !c.StartTime().Equal(want) {
		t.Errorf("StartTime = %v, want %v", c.StartTime(), want)
	}
}

func TestUpdate_Description(t *testing.T) {
	var nilDescription *string
	tests := []struct {
		name      string
		arg       **string
		wantField bool
		want      *string
	}{
		{"a pointer to a text replaces it", ptr(ptr("new")), false, ptr("new")},
		{"a pointer to nil clears it", &nilDescription, false, nil},
		{"nil leaves it alone", nil, false, ptr("old")},
		{"exactly 5000 multi-byte characters are accepted", ptr(ptr(strings.Repeat("é", 5000))), false, ptr(strings.Repeat("é", 5000))},
		{"5001 characters are rejected and nothing changes", ptr(ptr(strings.Repeat("a", 5001))), true, ptr("old")},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := contestWithDescription(t, ptr("old"))

			err := updateArgs{description: tt.arg}.apply(c, fixedNow)

			if tt.wantField {
				assertValidationField(t, tt.name, err, "description")
			} else if err != nil {
				t.Fatalf("Update: %v", err)
			}
			got := c.Description()
			switch {
			case tt.want == nil && got != nil:
				t.Errorf("Description = %q, want nil", *got)
			case tt.want != nil && (got == nil || *got != *tt.want):
				t.Errorf("Description mismatch, want %d characters", len([]rune(*tt.want)))
			}
		})
	}
}

func TestUpdate_Rejections_LeaveTheContestUntouched(t *testing.T) {
	// The contest runs fixedNow+1h .. fixedNow+4h (180 minutes) with a 60 minute freeze.
	rename := ptr(mustName(t, "Should not stick"))
	tests := []struct {
		name      string
		args      updateArgs
		wantCode  string
		wantField string
	}{
		{"a start exactly now", updateArgs{start: ptr(fixedNow)}, contest.ErrCodeStartTimeInPast, ""},
		{"a start in the past", updateArgs{start: ptr(fixedNow.Add(-time.Minute))}, contest.ErrCodeStartTimeInPast, ""},
		{"an end exactly now", updateArgs{end: ptr(fixedNow)}, contest.ErrCodeEndTimeInPast, ""},
		{"an end in the past", updateArgs{end: ptr(fixedNow.Add(-time.Minute))}, contest.ErrCodeEndTimeInPast, ""},
		{"an end before the start, both given", updateArgs{start: ptr(fixedNow.Add(3 * time.Hour)), end: ptr(fixedNow.Add(2 * time.Hour))}, contest.ErrCodeInvalidTimeRange, ""},
		{"only an end before the current start", updateArgs{end: ptr(fixedNow.Add(30 * time.Minute))}, contest.ErrCodeInvalidTimeRange, ""},
		{"only a start after the current end", updateArgs{start: ptr(fixedNow.Add(5 * time.Hour))}, contest.ErrCodeInvalidTimeRange, ""},
		{"only a start equal to the current end", updateArgs{start: ptr(fixedNow.Add(4 * time.Hour))}, contest.ErrCodeInvalidTimeRange, ""},
		{"a freeze as long as the contest", updateArgs{freeze: ptr(180)}, contest.ErrCodeFreezeTooLong, ""},
		{"a freeze as long as the new duration", updateArgs{end: ptr(fixedNow.Add(2 * time.Hour)), freeze: ptr(60)}, contest.ErrCodeFreezeTooLong, ""},
		{"an end that leaves the contest as short as its existing freeze", updateArgs{end: ptr(fixedNow.Add(2 * time.Hour))}, contest.ErrCodeFreezeTooLong, ""},
		{"an end that leaves the contest shorter than its existing freeze", updateArgs{end: ptr(fixedNow.Add(90 * time.Minute))}, contest.ErrCodeFreezeTooLong, ""},
		{"a later start that leaves the contest shorter than its existing freeze", updateArgs{start: ptr(fixedNow.Add(3*time.Hour + 30*time.Minute))}, contest.ErrCodeFreezeTooLong, ""},
		{"a negative freeze", updateArgs{freeze: ptr(-1)}, "", "freezeMinutes"},
		{"a description over the limit", updateArgs{description: ptr(ptr(strings.Repeat("a", 5001)))}, "", "description"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := validContest(t)
			before := fieldsOf(c)
			tt.args.name = rename

			err := tt.args.apply(c, fixedNow)

			if tt.wantField != "" {
				assertValidationField(t, tt.name, err, tt.wantField)
			} else {
				assertErrorCode(t, tt.name, err, tt.wantCode)
			}
			if after := fieldsOf(c); !reflect.DeepEqual(before, after) {
				t.Errorf("a rejected update changed the contest: before %+v, after %+v", before, after)
			}
		})
	}
}

func TestUpdate_AcceptsTheBoundaries(t *testing.T) {
	tests := []struct {
		name  string
		args  updateArgs
		check func(t *testing.T, c *contest.Contest)
	}{
		{"a freeze one minute shorter than the contest", updateArgs{freeze: ptr(179)}, func(t *testing.T, c *contest.Contest) {
			if c.FreezeMinutes() != 179 {
				t.Errorf("FreezeMinutes = %d, want 179", c.FreezeMinutes())
			}
		}},
		{"a freeze of zero removes it", updateArgs{freeze: ptr(0)}, func(t *testing.T, c *contest.Contest) {
			if c.FreezeMinutes() != 0 {
				t.Errorf("FreezeMinutes = %d, want 0", c.FreezeMinutes())
			}
		}},
		{"a start one second after now", updateArgs{start: ptr(fixedNow.Add(time.Second))}, func(t *testing.T, c *contest.Contest) {
			if want := fixedNow.Add(time.Second); !c.StartTime().Equal(want) {
				t.Errorf("StartTime = %v, want %v", c.StartTime(), want)
			}
		}},
		{"an end one minute past the freeze length", updateArgs{start: ptr(fixedNow.Add(2 * time.Hour)), end: ptr(fixedNow.Add(3*time.Hour + time.Minute))}, func(t *testing.T, c *contest.Contest) {
			if c.Duration() != 61 {
				t.Errorf("Duration = %d, want 61", c.Duration())
			}
		}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := validContest(t)

			err := tt.args.apply(c, fixedNow)

			if err != nil {
				t.Fatalf("Update: %v", err)
			}
			tt.check(t, c)
		})
	}
}

func TestContestSetters_StoreTheValueAndTouchUpdatedAtInUTC(t *testing.T) {
	now := time.Date(2026, 5, 4, 15, 0, 0, 0, time.FixedZone("UTC+2", 2*3600))
	wantUpdatedAt := time.Date(2026, 5, 4, 13, 0, 0, 0, time.UTC)
	tests := []struct {
		name  string
		set   func(c *contest.Contest)
		check func(t *testing.T, c *contest.Contest)
	}{
		{"participation mode", func(c *contest.Contest) {
			c.SetParticipationMode(contest.ParticipationModeTeam, now)
		}, func(t *testing.T, c *contest.Contest) {
			if c.ParticipationMode().String() != "TEAM" {
				t.Errorf("ParticipationMode = %s, want TEAM", c.ParticipationMode())
			}
		}},
		{"team size", func(c *contest.Contest) {
			c.SetTeamSize(contest.RestoreTeamSize(3, 4), now)
		}, func(t *testing.T, c *contest.Contest) {
			if c.TeamSize().Min() != 3 || c.TeamSize().Max() != 4 {
				t.Errorf("TeamSize = %d..%d, want 3..4", c.TeamSize().Min(), c.TeamSize().Max())
			}
		}},
		{"show team members", func(c *contest.Contest) {
			c.SetShowTeamMembers(true, now)
		}, func(t *testing.T, c *contest.Contest) {
			if !c.ShowTeamMembers() {
				t.Error("ShowTeamMembers = false, want true")
			}
		}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := validContest(t)

			tt.set(c)

			tt.check(t, c)
			if c.UpdatedAt() == nil || !c.UpdatedAt().Equal(wantUpdatedAt) || c.UpdatedAt().Location() != time.UTC {
				t.Errorf("UpdatedAt = %v, want %v in UTC", c.UpdatedAt(), wantUpdatedAt)
			}
		})
	}
}

func TestUpdate_ShorteningTheContest_NeedsAFreezeThatStillFits(t *testing.T) {
	// The contest runs fixedNow+1h .. fixedNow+4h with a 60 minute freeze.
	end90 := fixedNow.Add(90 * time.Minute)
	tests := []struct {
		name       string
		args       updateArgs
		wantFreeze int
		wantMins   int
	}{
		{"removing the freeze in the same request", updateArgs{end: ptr(end90), freeze: ptr(0)}, 0, 30},
		{"lowering the freeze in the same request", updateArgs{end: ptr(end90), freeze: ptr(10)}, 10, 30},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := validContest(t)

			err := tt.args.apply(c, fixedNow)

			if err != nil {
				t.Fatalf("Update: %v", err)
			}
			if c.FreezeMinutes() != tt.wantFreeze || c.Duration() != tt.wantMins {
				t.Errorf("freeze/duration = %d/%d, want %d/%d", c.FreezeMinutes(), c.Duration(), tt.wantFreeze, tt.wantMins)
			}
		})
	}
}

func TestUpdate_ContestAlreadyInconsistent_CanStillBeRenamed(t *testing.T) {
	start := fixedNow.Add(time.Hour)
	c := contest.RestoreContest(
		"contest-id",
		mustName(t, "Legacy"),
		nil,
		start, start.Add(time.Hour),
		mustPenalty(t, 20),
		120,
		false, false, false, false,
		nil, nil,
		shared.RestoreGroupID("group-id"),
		shared.RestoreUserID("owner-id"),
		contest.RestoreParticipationMode("INDIVIDUAL"), contest.RestoreTeamSize(2, 5),
		nil,
		fixedNow,
		nil,
	)

	err := updateArgs{name: ptr(mustName(t, "Renamed"))}.apply(c, fixedNow)

	if err != nil {
		t.Fatalf("Update: %v", err)
	}
	if c.Name().Value() != "Renamed" {
		t.Errorf("Name = %q, want Renamed", c.Name().Value())
	}
}
