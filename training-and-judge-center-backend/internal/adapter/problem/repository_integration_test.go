//go:build integration

package problem

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/training-judge-center/backend/internal/adapter/postgres"
	domainProblem "github.com/training-judge-center/backend/internal/domain/problem"
	"github.com/training-judge-center/backend/internal/domain/shared"
	"github.com/training-judge-center/backend/internal/testutil/pgtest"
	"github.com/training-judge-center/backend/pkg/apperror"
)

var integrationNow = time.Date(2026, 8, 8, 12, 0, 0, 0, time.UTC)

const (
	authorA = "11111111-1111-1111-1111-111111111111"
	authorB = "22222222-2222-2222-2222-222222222222"
	viewerC = "33333333-3333-3333-3333-333333333333"
	idOne   = "aaaaaaaa-0000-0000-0000-000000000001"
	idTwo   = "aaaaaaaa-0000-0000-0000-000000000002"
	idThree = "aaaaaaaa-0000-0000-0000-000000000003"
)

func newTestProblem(t *testing.T, id, slug, title, author string, tags ...string) *domainProblem.Problem {
	t.Helper()
	s, err := domainProblem.NewSlug(slug)
	require.NoError(t, err)
	ti, err := domainProblem.NewTitle(title)
	require.NoError(t, err)
	body := "Statement of " + title
	st, err := domainProblem.NewStatement(&body)
	require.NoError(t, err)
	authorID, err := shared.NewUserID(author)
	require.NoError(t, err)
	p, err := domainProblem.NewProblem(id, s, ti, st, nil, nil, nil, domainProblem.RestoreTags(tags), authorID, integrationNow.Add(time.Duration(id[len(id)-1]-'0')*time.Minute))
	require.NoError(t, err)
	return p
}

func seedProblem(t *testing.T, pool *pgxpool.Pool, p *domainProblem.Problem) *domainProblem.Problem {
	t.Helper()
	require.NoError(t, NewRepository(pool).Save(context.Background(), p))
	return p
}

func slugOf(t *testing.T, value string) domainProblem.Slug {
	t.Helper()
	s, err := domainProblem.NewSlug(value)
	require.NoError(t, err)
	return s
}

func requireProblemNotFound(t *testing.T, err error) {
	t.Helper()
	var appErr *apperror.AppError
	require.ErrorAs(t, err, &appErr)
	assert.Equal(t, domainProblem.ErrCodeProblemNotFound, appErr.Code)
}

func slugs(problems []*domainProblem.Problem) []string {
	out := make([]string, len(problems))
	for i, p := range problems {
		out[i] = p.Slug().String()
	}
	return out
}

func TestRepository_Save_RoundTripsEveryField(t *testing.T) {
	pool := pgtest.NewDB(t)
	repo := NewRepository(pool)
	p := newTestProblem(t, idOne, "two-sum", "Two Sum", authorA, "arrays", "hashing")
	tl, err := domainProblem.NewTimeLimit(2000, 10000)
	require.NoError(t, err)
	ml, err := domainProblem.NewMemoryLimit(256, 1024)
	require.NoError(t, err)
	limit, mem := 3000, 512
	p.UpdateMetadata(nil, nil, &tl, &ml, []domainProblem.LanguageOverride{domainProblem.RestoreLanguageOverride("python", &limit, &mem)}, nil, integrationNow)
	p.SetTestCases("problems/two-sum/tests.zip", integrationNow)
	p.AddSolution(domainProblem.RestoreJudgingFile("sol.cpp", "problems/two-sum/sol.cpp", "cpp", nil, nil), integrationNow)
	compiledKey := "problems/two-sum/chk.bin"
	compiledAt := integrationNow.Add(time.Minute)
	p.SetChecker(domainProblem.RestoreJudgingFile("chk.cpp", "problems/two-sum/chk.cpp", "cpp", &compiledKey, &compiledAt), integrationNow)
	p.SetValidator(domainProblem.RestoreJudgingFile("val.cpp", "problems/two-sum/val.cpp", "cpp", nil, nil), integrationNow)
	p.UpdateAccessibility(domainProblem.NewAccessibilityPublic(), integrationNow)

	require.NoError(t, repo.Save(context.Background(), p))

	got, err := repo.FindBySlug(context.Background(), slugOf(t, "two-sum"))
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, idOne, got.ID())
	assert.Equal(t, "Two Sum", got.Title().String())
	require.NotNil(t, got.Statement().Value())
	assert.Equal(t, "Statement of Two Sum", *got.Statement().Value())
	assert.Equal(t, 2000, got.TimeLimit().Milliseconds())
	assert.Equal(t, 256, got.MemoryLimit().Megabytes())
	assert.Equal(t, []string{"arrays", "hashing"}, got.Tags().Values())
	assert.Equal(t, "DRAFT", got.Status().String())
	assert.Equal(t, "PUBLIC", got.Accessibility().String())
	assert.Equal(t, authorA, got.AuthorID().String())
	require.NotNil(t, got.TestCasesKey())
	assert.Equal(t, "problems/two-sum/tests.zip", *got.TestCasesKey())
	require.Len(t, got.LangOverrides(), 1)
	assert.Equal(t, "python", got.LangOverrides()[0].Language())
	assert.Equal(t, 3000, *got.LangOverrides()[0].TimeLimit())
	assert.Equal(t, 512, *got.LangOverrides()[0].MemoryLimit())
	require.Len(t, got.Solutions(), 1)
	assert.Equal(t, "sol.cpp", got.Solutions()[0].Filename())
	require.NotNil(t, got.Checker())
	assert.Equal(t, "chk.cpp", got.Checker().Filename())
	require.NotNil(t, got.Checker().CompiledKey())
	assert.Equal(t, "problems/two-sum/chk.bin", *got.Checker().CompiledKey())
	require.NotNil(t, got.Checker().CompiledAt())
	assert.True(t, compiledAt.Equal(*got.Checker().CompiledAt()))
	require.NotNil(t, got.Validator())
	assert.Equal(t, "val.cpp", got.Validator().Filename())
	require.NotNil(t, got.JudgingUpdatedAt())
}

func TestRepository_Save_ExistingIDUpdatesInPlace(t *testing.T) {
	pool := pgtest.NewDB(t)
	repo := NewRepository(pool)
	p := seedProblem(t, pool, newTestProblem(t, idOne, "two-sum", "Two Sum", authorA))

	require.NoError(t, p.Publish(integrationNow.Add(time.Hour)))
	newTitle, err := domainProblem.NewTitle("Two Sum Revisited")
	require.NoError(t, err)
	p.UpdateMetadata(&newTitle, nil, nil, nil, nil, nil, integrationNow.Add(time.Hour))
	require.NoError(t, repo.Save(context.Background(), p))

	got, err := repo.FindBySlug(context.Background(), slugOf(t, "two-sum"))
	require.NoError(t, err)
	assert.Equal(t, "Two Sum Revisited", got.Title().String())
	assert.Equal(t, "PUBLISHED", got.Status().String())
	var rows int
	require.NoError(t, pool.QueryRow(context.Background(), `SELECT COUNT(*) FROM problems`).Scan(&rows))
	assert.Equal(t, 1, rows)
}

func TestRepository_Save_DuplicateSlugOnAnotherIDReturnsSlugConflict(t *testing.T) {
	pool := pgtest.NewDB(t)
	seedProblem(t, pool, newTestProblem(t, idOne, "two-sum", "Two Sum", authorA))

	err := NewRepository(pool).Save(context.Background(), newTestProblem(t, idTwo, "two-sum", "Other", authorB))

	var appErr *apperror.AppError
	require.ErrorAs(t, err, &appErr)
	assert.Equal(t, apperror.KindConflict, appErr.Kind)
}

func TestRepository_FindBySlug_MissingReturnsNotFound(t *testing.T) {
	pool := pgtest.NewDB(t)

	got, err := NewRepository(pool).FindBySlug(context.Background(), slugOf(t, "nope"))

	requireProblemNotFound(t, err)
	assert.Nil(t, got)
}

func TestRepository_ExistsBySlug(t *testing.T) {
	pool := pgtest.NewDB(t)
	seedProblem(t, pool, newTestProblem(t, idOne, "two-sum", "Two Sum", authorA))
	repo := NewRepository(pool)

	exists, err := repo.ExistsBySlug(context.Background(), slugOf(t, "two-sum"))
	require.NoError(t, err)
	assert.True(t, exists)

	exists, err = repo.ExistsBySlug(context.Background(), slugOf(t, "other"))
	require.NoError(t, err)
	assert.False(t, exists)
}

func TestRepository_Delete_RemovesTheProblem(t *testing.T) {
	pool := pgtest.NewDB(t)
	seedProblem(t, pool, newTestProblem(t, idOne, "two-sum", "Two Sum", authorA))
	repo := NewRepository(pool)

	require.NoError(t, repo.Delete(context.Background(), idOne))

	got, err := repo.FindBySlug(context.Background(), slugOf(t, "two-sum"))
	requireProblemNotFound(t, err)
	assert.Nil(t, got)
}

func TestRepository_Delete_MissingReturnsNotFound(t *testing.T) {
	pool := pgtest.NewDB(t)

	err := NewRepository(pool).Delete(context.Background(), idOne)

	var appErr *apperror.AppError
	require.ErrorAs(t, err, &appErr)
	assert.Equal(t, domainProblem.ErrCodeProblemNotFound, appErr.Code)
}

func seedCatalog(t *testing.T, pool *pgxpool.Pool) {
	t.Helper()
	published := seedProblem(t, pool, newTestProblem(t, idOne, "public-published", "Alpha Graph", authorA, "graphs"))
	require.NoError(t, published.Publish(integrationNow))
	published.UpdateAccessibility(domainProblem.NewAccessibilityPublic(), integrationNow)
	require.NoError(t, NewRepository(pool).Save(context.Background(), published))

	draftA := newTestProblem(t, idTwo, "draft-of-a", "Beta Draft", authorA, "graphs", "dp")
	draftA.RemoveModifier(shared.RestoreUserID(authorA), integrationNow)
	require.NoError(t, draftA.AddModifier(shared.RestoreUserID(viewerC), integrationNow))
	seedProblem(t, pool, draftA)

	seedProblem(t, pool, newTestProblem(t, idThree, "draft-of-b", "Gamma 100%", authorB, "dp"))
}

func listFilters(mutate func(f *domainProblem.ListFilters)) domainProblem.ListFilters {
	f := domainProblem.ListFilters{Page: 1, Limit: 10}
	mutate(&f)
	return f
}

func TestRepository_List_FiltersSearchesAndPaginates(t *testing.T) {
	pool := pgtest.NewDB(t)
	seedCatalog(t, pool)
	repo := NewRepository(pool)
	authorAID := authorA
	viewerCID := viewerC
	authorBID := authorB
	public := domainProblem.NewAccessibilityPublic()
	published := domainProblem.NewStatusPublished()
	draft := domainProblem.NewStatusDraft()

	tests := []struct {
		name      string
		mutate    func(f *domainProblem.ListFilters)
		wantSlugs []string
		wantTotal int
	}{
		{"empty statuses means no status filter", func(f *domainProblem.ListFilters) {}, []string{"draft-of-b", "draft-of-a", "public-published"}, 3},
		{"statuses published only", func(f *domainProblem.ListFilters) { f.Statuses = []domainProblem.Status{published} }, []string{"public-published"}, 1},
		{"statuses draft only", func(f *domainProblem.ListFilters) { f.Statuses = []domainProblem.Status{draft} }, []string{"draft-of-b", "draft-of-a"}, 2},
		{"viewer sees public published plus own drafts and modified ones", func(f *domainProblem.ListFilters) { f.ViewerModifierID = &viewerCID }, []string{"draft-of-a", "public-published"}, 2},
		{"author viewer sees own drafts but not another author's draft", func(f *domainProblem.ListFilters) { f.ViewerModifierID = &authorAID }, []string{"draft-of-a", "public-published"}, 2},
		{"viewer without any link sees only the public published", func(f *domainProblem.ListFilters) {
			id := "99999999-9999-9999-9999-999999999999"
			f.ViewerModifierID = &id
		}, []string{"public-published"}, 1},
		{"author filter", func(f *domainProblem.ListFilters) { f.AuthorID = &authorBID }, []string{"draft-of-b"}, 1},
		{"tag contains", func(f *domainProblem.ListFilters) { f.Tags = []string{"graphs"} }, []string{"draft-of-a", "public-published"}, 2},
		{"all tags required", func(f *domainProblem.ListFilters) { f.Tags = []string{"graphs", "dp"} }, []string{"draft-of-a"}, 1},
		{"accessibility", func(f *domainProblem.ListFilters) { f.Accessibility = &public }, []string{"public-published"}, 1},
		{"search is case insensitive", func(f *domainProblem.ListFilters) { f.Search = "alpha" }, []string{"public-published"}, 1},
		{"percent in search is literal", func(f *domainProblem.ListFilters) { f.Search = "%" }, []string{"draft-of-b"}, 1},
		{"page two of size two", func(f *domainProblem.ListFilters) { f.Page = 2; f.Limit = 2 }, []string{"public-published"}, 3},
		{"no match is an empty slice not nil", func(f *domainProblem.ListFilters) { f.Search = "zzz" }, []string{}, 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, total, err := repo.List(context.Background(), listFilters(tt.mutate))

			require.NoError(t, err)
			assert.Equal(t, tt.wantTotal, total)
			assert.Equal(t, tt.wantSlugs, slugs(got))
		})
	}
}

func TestRepository_List_OmitsTheStatement(t *testing.T) {
	pool := pgtest.NewDB(t)
	seedCatalog(t, pool)

	got, _, err := NewRepository(pool).List(context.Background(), listFilters(func(f *domainProblem.ListFilters) {}))

	require.NoError(t, err)
	require.NotEmpty(t, got)
	for _, p := range got {
		assert.Nil(t, p.Statement().Value())
	}
}

func TestRepository_InsideTransaction_ReadsSeeTheWriteAndRollbackDiscardsIt(t *testing.T) {
	pool := pgtest.NewDB(t)
	repo := NewRepository(pool)
	ctx := context.Background()
	rollback := errors.New("force rollback")

	err := postgres.NewTransactionManager(pool).WithTx(ctx, func(txCtx context.Context) error {
		require.NoError(t, repo.Save(txCtx, newTestProblem(t, idOne, "two-sum", "Two Sum", authorA)))

		inside, err := repo.FindBySlug(txCtx, slugOf(t, "two-sum"))
		require.NoError(t, err)
		assert.NotNil(t, inside)
		exists, err := repo.ExistsBySlug(txCtx, slugOf(t, "two-sum"))
		require.NoError(t, err)
		assert.True(t, exists)
		return rollback
	})

	require.ErrorIs(t, err, rollback)
	after, err := repo.FindBySlug(ctx, slugOf(t, "two-sum"))
	requireProblemNotFound(t, err)
	assert.Nil(t, after)
}
