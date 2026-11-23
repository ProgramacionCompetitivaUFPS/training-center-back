//go:build integration

package submission

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/training-judge-center/backend/internal/domain/shared"
	domainSubmission "github.com/training-judge-center/backend/internal/domain/submission"
	"github.com/training-judge-center/backend/internal/testutil/pgtest"
	"github.com/training-judge-center/backend/pkg/apperror"
)

var itNow = time.Date(2026, 8, 8, 12, 0, 0, 0, time.UTC)

const (
	itUserA    = "11111111-1111-1111-1111-111111111111"
	itUserB    = "22222222-2222-2222-2222-222222222222"
	itProblem1 = "aaaaaaaa-0000-0000-0000-000000000001"
	itProblem2 = "aaaaaaaa-0000-0000-0000-000000000002"
)

func itSeedBase(t *testing.T, pool *pgxpool.Pool) {
	t.Helper()
	pgtest.SeedUser(t, pool, itUserA, "ana")
	pgtest.SeedUser(t, pool, itUserB, "beto")
	pgtest.SeedProblem(t, pool, itProblem1, "two-sum", "PUBLISHED")
	pgtest.SeedProblem(t, pool, itProblem2, "three-sum", "PUBLISHED")
}

func itSubmission(t *testing.T, id, problemID, userID string, at time.Time) *domainSubmission.Submission {
	t.Helper()
	lang, err := domainSubmission.NewLanguage("cpp20")
	require.NoError(t, err)
	s, err := domainSubmission.NewSubmission(id, problemID, shared.RestoreUserID(userID), nil, nil, lang, "g++ 13", "src/"+id, "hash-"+id, 120, "Two Sum", "two-sum", at)
	require.NoError(t, err)
	return s
}

func itSave(t *testing.T, pool *pgxpool.Pool, s *domainSubmission.Submission) {
	t.Helper()
	require.NoError(t, NewRepository(pool).Save(context.Background(), s))
}

func itStatusOf(t *testing.T, pool *pgxpool.Pool, id string) string {
	t.Helper()
	var status string
	require.NoError(t, pool.QueryRow(context.Background(), `SELECT status FROM submissions WHERE id = $1`, id).Scan(&status))
	return status
}

func itIDs(subs []*domainSubmission.Submission) []string {
	out := make([]string, len(subs))
	for i, s := range subs {
		out[i] = s.ID()
	}
	return out
}

func TestRepository_Save_RoundTripsEveryField(t *testing.T) {
	pool := pgtest.NewDB(t)
	itSeedBase(t, pool)
	s := itSubmission(t, "sub-1", itProblem1, itUserA, itNow)
	s.AttachBlocklyWorkspace("ws/sub-1.xml")
	itSave(t, pool, s)

	got, err := NewRepository(pool).FindByID(context.Background(), "sub-1")

	require.NoError(t, err)
	assert.Equal(t, itProblem1, got.ProblemID())
	assert.Equal(t, itUserA, got.UserID().String())
	assert.Equal(t, "cpp20", got.Language().String())
	assert.Equal(t, "g++ 13", got.Compiler())
	assert.Equal(t, "PENDING", got.Status().String())
	assert.Equal(t, "PRIVATE", got.Visibility().String())
	assert.Equal(t, "src/sub-1", got.SourceCodePath())
	assert.Equal(t, "hash-sub-1", got.FileHash())
	assert.Equal(t, 120, got.FileSize())
	assert.Equal(t, "Two Sum", got.ProblemTitle())
	assert.Equal(t, "two-sum", got.ProblemSlug())
	require.NotNil(t, got.WorkspaceXMLPath())
	assert.Equal(t, "ws/sub-1.xml", *got.WorkspaceXMLPath())
	assert.True(t, itNow.Equal(got.SubmittedAt()))
	assert.Nil(t, got.ContestID())
	assert.Nil(t, got.JudgedAt())
}

func TestRepository_Save_ExistingIDUpdatesMutableFields(t *testing.T) {
	pool := pgtest.NewDB(t)
	itSeedBase(t, pool)
	s := itSubmission(t, "sub-1", itProblem1, itUserA, itNow)
	itSave(t, pool, s)

	s.SetVisibility(domainSubmission.NewVisibilityPublic())
	itSave(t, pool, s)

	got, err := NewRepository(pool).FindByID(context.Background(), "sub-1")
	require.NoError(t, err)
	assert.Equal(t, "PUBLIC", got.Visibility().String())
	var rows int
	require.NoError(t, pool.QueryRow(context.Background(), `SELECT COUNT(*) FROM submissions`).Scan(&rows))
	assert.Equal(t, 1, rows)
}

func TestRepository_Save_UnknownProblemReturnsInternalNotRawError(t *testing.T) {
	pool := pgtest.NewDB(t)
	itSeedBase(t, pool)

	err := NewRepository(pool).Save(context.Background(), itSubmission(t, "sub-1", "aaaaaaaa-0000-0000-0000-0000000000ff", itUserA, itNow))

	assertAppErrorKind(t, err, apperror.KindInternal)
}

func TestRepository_FindByID_MissingReturnsNotFound(t *testing.T) {
	pool := pgtest.NewDB(t)

	got, err := NewRepository(pool).FindByID(context.Background(), "nope")

	assertAppErrorCode(t, err, domainSubmission.ErrCodeSubmissionNotFound)
	assert.Nil(t, got)
}

func TestRepository_Update_PersistsTheVerdict(t *testing.T) {
	pool := pgtest.NewDB(t)
	itSeedBase(t, pool)
	s := itSubmission(t, "sub-1", itProblem1, itUserA, itNow)
	itSave(t, pool, s)
	memory := 2048
	require.NoError(t, s.Start(itNow.Add(time.Second)))
	require.NoError(t, s.MarkAccepted(150, &memory, itNow.Add(2*time.Second)))

	require.NoError(t, NewRepository(pool).Update(context.Background(), s))

	got, err := NewRepository(pool).FindByID(context.Background(), "sub-1")
	require.NoError(t, err)
	assert.Equal(t, "ACCEPTED", got.Status().String())
	require.NotNil(t, got.TimeMs())
	assert.Equal(t, 150, *got.TimeMs())
	require.NotNil(t, got.MemoryKb())
	assert.Equal(t, 2048, *got.MemoryKb())
	require.NotNil(t, got.JudgedAt())
}

func TestRepository_Update_MissingReturnsNotFound(t *testing.T) {
	pool := pgtest.NewDB(t)
	itSeedBase(t, pool)

	err := NewRepository(pool).Update(context.Background(), itSubmission(t, "ghost", itProblem1, itUserA, itNow))

	assertAppErrorCode(t, err, domainSubmission.ErrCodeSubmissionNotFound)
}

func TestRepository_UpdateVisibility(t *testing.T) {
	pool := pgtest.NewDB(t)
	itSeedBase(t, pool)
	s := itSubmission(t, "sub-1", itProblem1, itUserA, itNow)
	itSave(t, pool, s)
	repo := NewRepository(pool)

	s.SetVisibility(domainSubmission.NewVisibilityPublic())
	require.NoError(t, repo.UpdateVisibility(context.Background(), s))

	got, err := repo.FindByID(context.Background(), "sub-1")
	require.NoError(t, err)
	assert.Equal(t, "PUBLIC", got.Visibility().String())

	err = repo.UpdateVisibility(context.Background(), itSubmission(t, "ghost", itProblem1, itUserA, itNow))
	assertAppErrorCode(t, err, domainSubmission.ErrCodeSubmissionNotFound)
}

func TestRepository_FindLastByUserAndProblem_ReturnsTheNewestWithoutContest(t *testing.T) {
	pool := pgtest.NewDB(t)
	itSeedBase(t, pool)
	itSave(t, pool, itSubmission(t, "old", itProblem1, itUserA, itNow))
	itSave(t, pool, itSubmission(t, "new", itProblem1, itUserA, itNow.Add(time.Hour)))
	itSave(t, pool, itSubmission(t, "other-problem", itProblem2, itUserA, itNow.Add(2*time.Hour)))
	itSave(t, pool, itSubmission(t, "other-user", itProblem1, itUserB, itNow.Add(3*time.Hour)))
	repo := NewRepository(pool)

	got, err := repo.FindLastByUserAndProblem(context.Background(), itUserA, itProblem1, nil)

	require.NoError(t, err)
	assert.Equal(t, "new", got.ID())

	_, err = repo.FindLastByUserAndProblem(context.Background(), itUserB, itProblem2, nil)
	assertAppErrorCode(t, err, domainSubmission.ErrCodeSubmissionNotFound)
}

func TestRepository_ExistsByHashAndUserAndProblem(t *testing.T) {
	pool := pgtest.NewDB(t)
	itSeedBase(t, pool)
	itSave(t, pool, itSubmission(t, "sub-1", itProblem1, itUserA, itNow))
	repo := NewRepository(pool)

	tests := []struct {
		name                  string
		hash, user, problemID string
		want                  bool
	}{
		{"same hash user and problem", "hash-sub-1", itUserA, itProblem1, true},
		{"other hash", "other", itUserA, itProblem1, false},
		{"other user", "hash-sub-1", itUserB, itProblem1, false},
		{"other problem", "hash-sub-1", itUserA, itProblem2, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := repo.ExistsByHashAndUserAndProblem(context.Background(), tt.hash, tt.user, tt.problemID, nil)

			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestRepository_List_FiltersAndPaginates(t *testing.T) {
	pool := pgtest.NewDB(t)
	itSeedBase(t, pool)
	repo := NewRepository(pool)
	a1 := itSubmission(t, "a1", itProblem1, itUserA, itNow)
	a2 := itSubmission(t, "a2", itProblem2, itUserA, itNow.Add(time.Hour))
	b1 := itSubmission(t, "b1", itProblem1, itUserB, itNow.Add(2*time.Hour))
	b1.SetVisibility(domainSubmission.NewVisibilityPublic())
	itSave(t, pool, a1)
	itSave(t, pool, a2)
	itSave(t, pool, b1)
	_, err := pool.Exec(context.Background(), `UPDATE submissions SET status = 'ACCEPTED' WHERE id = 'a2'`)
	require.NoError(t, err)
	_, err = pool.Exec(context.Background(), `UPDATE submissions SET language = 'python310', workspace_xml_path = 'ws/a1.xml' WHERE id = 'a1'`)
	require.NoError(t, err)

	userA := shared.RestoreUserID(itUserA)
	userB := shared.RestoreUserID(itUserB)
	problem1 := itProblem1
	accepted := "ACCEPTED"
	python := "python310"
	public := "PUBLIC"
	yes, no := true, false
	from := itNow.Add(30 * time.Minute)
	before := itNow.Add(90 * time.Minute)

	tests := []struct {
		name      string
		filters   domainSubmission.ListFilters
		wantIDs   []string
		wantTotal int
	}{
		{"no filter newest first", domainSubmission.ListFilters{}, []string{"b1", "a2", "a1"}, 3},
		{"by user", domainSubmission.ListFilters{UserID: &userA}, []string{"a2", "a1"}, 2},
		{"by problem", domainSubmission.ListFilters{ProblemID: &problem1}, []string{"b1", "a1"}, 2},
		{"by status", domainSubmission.ListFilters{Status: &accepted}, []string{"a2"}, 1},
		{"by language", domainSubmission.ListFilters{Language: &python}, []string{"a1"}, 1},
		{"by visibility", domainSubmission.ListFilters{Visibility: &public}, []string{"b1"}, 1},
		{"owner sees public plus own", domainSubmission.ListFilters{OwnerID: &userA}, []string{"b1", "a2", "a1"}, 3},
		{"owner b sees only own and public", domainSubmission.ListFilters{OwnerID: &userB, UserID: &userA}, []string{}, 0},
		{"blockly only", domainSubmission.ListFilters{Blockly: &yes}, []string{"a1"}, 1},
		{"no blockly", domainSubmission.ListFilters{Blockly: &no}, []string{"b1", "a2"}, 2},
		{"exclude contest keeps practice", domainSubmission.ListFilters{ExcludeContest: true}, []string{"b1", "a2", "a1"}, 3},
		{"submitted window", domainSubmission.ListFilters{SubmittedFrom: &from, SubmittedBefore: &before}, []string{"a2"}, 1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.filters.Page, tt.filters.Limit = 1, 10

			got, total, err := repo.List(context.Background(), tt.filters)

			require.NoError(t, err)
			assert.Equal(t, tt.wantTotal, total)
			assert.Equal(t, tt.wantIDs, itIDs(got))
		})
	}

	page, total, err := repo.List(context.Background(), domainSubmission.ListFilters{Page: 2, Limit: 2})
	require.NoError(t, err)
	assert.Equal(t, 3, total)
	assert.Equal(t, []string{"a1"}, itIDs(page))
}

func TestRepository_ClaimForJudging_OnlyOneConcurrentWorkerWins(t *testing.T) {
	pool := pgtest.NewDB(t)
	itSeedBase(t, pool)
	itSave(t, pool, itSubmission(t, "sub-1", itProblem1, itUserA, itNow))
	repo := NewRepository(pool)

	var wins atomic.Int32
	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			claimed, err := repo.ClaimForJudging(context.Background(), "sub-1")
			if err == nil && claimed {
				wins.Add(1)
			}
		}()
	}
	wg.Wait()

	assert.Equal(t, int32(1), wins.Load())
	assert.Equal(t, "RUNNING", itStatusOf(t, pool, "sub-1"))
}

func TestRepository_ClaimForJudging_NotPendingOrMissingIsNotClaimed(t *testing.T) {
	pool := pgtest.NewDB(t)
	itSeedBase(t, pool)
	pgtest.SeedSubmission(t, pool, pgtest.Submission{ID: "done", ProblemID: itProblem1, UserID: itUserA, Status: "ACCEPTED"})
	repo := NewRepository(pool)

	claimed, err := repo.ClaimForJudging(context.Background(), "done")
	require.NoError(t, err)
	assert.False(t, claimed)
	assert.Equal(t, "ACCEPTED", itStatusOf(t, pool, "done"))

	claimed, err = repo.ClaimForJudging(context.Background(), "ghost")
	require.NoError(t, err)
	assert.False(t, claimed)
}
