package contest

import (
	"context"
	"errors"
	"testing"

	domainContest "github.com/training-judge-center/backend/internal/domain/contest"
	"github.com/training-judge-center/backend/pkg/apperror"
)

const (
	colorProblemA = "dddddddd-0000-0000-0000-00000000000a"
	colorProblemB = "dddddddd-0000-0000-0000-00000000000b"
)

func ptr[T any](v T) *T { return &v }

func colorChange(c *string) **string { return &c }

func problemWithColor(id, problemID string, order int, color string) domainContest.ContestProblem {
	c := domainContest.RestoreProblemColor(color)
	return domainContest.RestoreContestProblem(id, problemID, order, &c)
}

func contestHolding(problems ...domainContest.ContestProblem) *domainContest.Contest {
	c := newTestContest(callerID)
	c.SetProblems(problems, testNow)
	return c
}

func slugProblemProvider() *mockProblemProvider {
	ids := map[string]string{"problem-a": colorProblemA, "problem-b": colorProblemB}
	return &mockProblemProvider{
		findBySlugsFn: func(_ context.Context, slugs []string, _ string, _ bool) (map[string]*ProblemInfo, error) {
			result := make(map[string]*ProblemInfo, len(slugs))
			for _, s := range slugs {
				result[s] = &ProblemInfo{ID: ids[s], Slug: s, Title: s + "-title", IsPublished: true, CanAdd: true}
			}
			return result, nil
		},
	}
}

func updateProblems(t *testing.T, c *domainContest.Contest, problems []ProblemOrderInput) (*domainContest.Contest, *ContestOutput, error) {
	t.Helper()
	var saved *domainContest.Contest
	repo := repoWith(c)
	repo.updateFn = func(_ context.Context, updated *domainContest.Contest) error {
		saved = updated
		return nil
	}
	uc := newUpdateContestUseCase(repo, groupFound(), isLead(), slugProblemProvider())

	out, err := uc.Execute(context.Background(), UpdateContestInput{
		CurrentUser: asCoach(callerID),
		GroupID:     testGroupID,
		ContestID:   testContestID,
		Problems:    &problems,
	})
	return saved, out, err
}

func colorOf(t *testing.T, c *domainContest.Contest, problemID string) *string {
	t.Helper()
	for _, cp := range c.Problems() {
		if cp.ProblemID() == problemID {
			if cp.Color() == nil {
				return nil
			}
			return ptr(cp.Color().Value())
		}
	}
	t.Fatalf("problem %s not found in saved contest", problemID)
	return nil
}

func assertColor(t *testing.T, label string, got *string, want string) {
	t.Helper()
	if got == nil || *got != want {
		t.Errorf("%s: color = %v, want %q", label, got, want)
	}
}

func assertNoColor(t *testing.T, label string, got *string) {
	t.Helper()
	if got != nil {
		t.Errorf("%s: color = %q, want none", label, *got)
	}
}

func TestUpdateContest_OmittedColorKeepsTheCurrentOne(t *testing.T) {
	c := contestHolding(problemWithColor("cp-a", colorProblemA, 1, "#aabbcc"))

	saved, out, err := updateProblems(t, c, []ProblemOrderInput{{Slug: "problem-a", Order: 1}})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	assertColor(t, "saved", colorOf(t, saved, colorProblemA), "#aabbcc")
	assertColor(t, "response", out.Problems[0].Color, "#aabbcc")
}

func TestUpdateContest_NullColorClearsTheCurrentOne(t *testing.T) {
	c := contestHolding(problemWithColor("cp-a", colorProblemA, 1, "#aabbcc"))

	saved, out, err := updateProblems(t, c, []ProblemOrderInput{{Slug: "problem-a", Order: 1, Color: colorChange(nil)}})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	assertNoColor(t, "saved", colorOf(t, saved, colorProblemA))
	assertNoColor(t, "response", out.Problems[0].Color)
}

func TestUpdateContest_ColorIsSavedLowercased(t *testing.T) {
	c := contestHolding()

	saved, out, err := updateProblems(t, c, []ProblemOrderInput{{Slug: "problem-a", Order: 1, Color: colorChange(ptr("#AABBCC"))}})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	assertColor(t, "saved", colorOf(t, saved, colorProblemA), "#aabbcc")
	assertColor(t, "response", out.Problems[0].Color, "#aabbcc")
}

func TestUpdateContest_ColorReplacesTheCurrentOne(t *testing.T) {
	c := contestHolding(problemWithColor("cp-a", colorProblemA, 1, "#aabbcc"))

	saved, _, err := updateProblems(t, c, []ProblemOrderInput{{Slug: "problem-a", Order: 1, Color: colorChange(ptr("#112233"))}})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	assertColor(t, "saved", colorOf(t, saved, colorProblemA), "#112233")
}

func TestUpdateContest_NewProblemWithoutColorHasNone(t *testing.T) {
	c := contestHolding(problemWithColor("cp-a", colorProblemA, 1, "#aabbcc"))

	saved, out, err := updateProblems(t, c, []ProblemOrderInput{
		{Slug: "problem-a", Order: 1},
		{Slug: "problem-b", Order: 2},
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	assertColor(t, "kept problem", colorOf(t, saved, colorProblemA), "#aabbcc")
	assertNoColor(t, "new problem", colorOf(t, saved, colorProblemB))
	assertNoColor(t, "new problem response", out.Problems[1].Color)
}

func TestUpdateContest_ColorFollowsTheProblemWhenItIsReordered(t *testing.T) {
	c := contestHolding(
		problemWithColor("cp-a", colorProblemA, 1, "#aabbcc"),
		problemWithColor("cp-b", colorProblemB, 2, "#112233"),
	)

	saved, _, err := updateProblems(t, c, []ProblemOrderInput{
		{Slug: "problem-b", Order: 1},
		{Slug: "problem-a", Order: 2},
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	assertColor(t, "problem a", colorOf(t, saved, colorProblemA), "#aabbcc")
	assertColor(t, "problem b", colorOf(t, saved, colorProblemB), "#112233")
}

func TestUpdateContest_DuplicateSlugKeepsTheFirstColor(t *testing.T) {
	c := contestHolding()

	saved, _, err := updateProblems(t, c, []ProblemOrderInput{
		{Slug: "problem-a", Order: 1, Color: colorChange(ptr("#aabbcc"))},
		{Slug: "problem-a", Order: 2, Color: colorChange(ptr("#112233"))},
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	assertColor(t, "saved", colorOf(t, saved, colorProblemA), "#aabbcc")
}

func TestUpdateContest_InvalidColorIsRejectedAndNothingIsSaved(t *testing.T) {
	for _, bad := range []string{"red", "#FFF", "#GGGGGG", ""} {
		t.Run(bad, func(t *testing.T) {
			c := contestHolding(problemWithColor("cp-a", colorProblemA, 1, "#aabbcc"))

			saved, _, err := updateProblems(t, c, []ProblemOrderInput{{Slug: "problem-a", Order: 1, Color: colorChange(ptr(bad))}})

			var appErr *apperror.AppError
			if !errors.As(err, &appErr) || appErr.Code != domainContest.ErrCodeInvalidProblemColor {
				t.Fatalf("expected %s, got %v", domainContest.ErrCodeInvalidProblemColor, err)
			}
			if saved != nil {
				t.Error("an invalid color must not reach the repository")
			}
		})
	}
}

func TestUpdateContest_ColorChangeOnAFinishedContestIsClosed(t *testing.T) {
	c := finishedContest()
	c.SetProblems([]domainContest.ContestProblem{problemWithColor("cp-a", colorProblemA, 1, "#aabbcc")}, testNow)

	_, _, err := updateProblems(t, c, []ProblemOrderInput{{Slug: "problem-a", Order: 1, Color: colorChange(ptr("#112233"))}})

	var appErr *apperror.AppError
	if !errors.As(err, &appErr) || appErr.Code != domainContest.ErrCodeContestProblemsLocked {
		t.Errorf("expected %s, got %v", domainContest.ErrCodeContestProblemsLocked, err)
	}
}

func TestUpdateContest_ProblemsLeftUntouchedKeepTheirColorInTheResponse(t *testing.T) {
	c := contestHolding(problemWithColor("cp-a", colorProblemA, 1, "#aabbcc"))
	uc := newUpdateContestUseCase(repoWith(c), groupFound(), isLead(), defaultProblemProvider())
	name := "Renamed"

	out, err := uc.Execute(context.Background(), UpdateContestInput{
		CurrentUser: asCoach(callerID),
		GroupID:     testGroupID,
		ContestID:   testContestID,
		Name:        &name,
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	assertColor(t, "response", out.Problems[0].Color, "#aabbcc")
}
