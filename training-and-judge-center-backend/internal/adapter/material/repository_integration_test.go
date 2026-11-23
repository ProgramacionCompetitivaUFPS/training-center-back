//go:build integration

package material

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	appMaterial "github.com/training-judge-center/backend/internal/application/material"
	"github.com/training-judge-center/backend/internal/domain/material"
	"github.com/training-judge-center/backend/internal/domain/shared"
	"github.com/training-judge-center/backend/internal/testutil/pgtest"
	"github.com/training-judge-center/backend/pkg/apperror"
)

func TestMain(m *testing.M) {
	os.Exit(pgtest.Main(m))
}

var itNow = time.Date(2026, 8, 8, 12, 0, 0, 0, time.UTC)

const (
	itUserA   = "11111111-1111-1111-1111-111111111111"
	itUserB   = "22222222-2222-2222-2222-222222222222"
	itUserC   = "33333333-3333-3333-3333-333333333333"
	itGroup1  = "bbbbbbbb-0000-0000-0000-000000000001"
	itGroup2  = "bbbbbbbb-0000-0000-0000-000000000002"
	itMat1    = "dddd0000-0000-0000-0000-000000000001"
	itMat2    = "dddd0000-0000-0000-0000-000000000002"
	itMat3    = "dddd0000-0000-0000-0000-000000000003"
	itMissing = "99999999-9999-9999-9999-999999999999"
)

func itSeedBase(t *testing.T, pool *pgxpool.Pool) {
	t.Helper()
	pgtest.SeedUser(t, pool, itUserA, "ana")
	pgtest.SeedUser(t, pool, itUserB, "beto")
	pgtest.SeedUser(t, pool, itUserC, "carla")
	pgtest.SeedGroup(t, pool, itGroup1, "Group A", itUserA)
	pgtest.SeedGroup(t, pool, itGroup2, "Group B", itUserA)
}

type matSpec struct {
	id, group, author, title, content string
	tags                              []string
	status                            string
	pinned                            bool
	publishedAt                       *time.Time
}

func itMaterial(spec matSpec) *material.Material {
	var pinnedAt *time.Time
	if spec.pinned {
		pinnedAt = &itNow
	}
	return material.RestoreMaterial(spec.id, spec.group, shared.RestoreUserID(spec.author), spec.title, spec.content,
		spec.tags, spec.status, spec.pinned, pinnedAt, itNow, itNow, spec.publishedAt)
}

func itSave(t *testing.T, pool *pgxpool.Pool, m *material.Material) {
	t.Helper()
	require.NoError(t, NewRepository(pool).Save(context.Background(), m))
}

func itRequireNotFound(t *testing.T, err error) {
	t.Helper()
	var appErr *apperror.AppError
	require.ErrorAs(t, err, &appErr)
	assert.Equal(t, material.ErrCodeMaterialNotFound, appErr.Code)
}

func itTimePtr(t time.Time) *time.Time { return &t }

func seedLibrary(t *testing.T, pool *pgxpool.Pool) {
	t.Helper()
	itSeedBase(t, pool)
	itSave(t, pool, itMaterial(matSpec{id: itMat1, group: itGroup1, author: itUserA, title: "Graph Algorithms", content: "BFS and DFS", tags: []string{"graphs"}, status: "PUBLISHED", pinned: true, publishedAt: itTimePtr(itNow)}))
	itSave(t, pool, itMaterial(matSpec{id: itMat2, group: itGroup1, author: itUserA, title: "Dynamic Programming", content: "A guide to algorithms for dp", tags: []string{"dp", "graphs"}, status: "PUBLISHED", publishedAt: itTimePtr(itNow.Add(48 * time.Hour))}))
	itSave(t, pool, itMaterial(matSpec{id: itMat3, group: itGroup1, author: itUserB, title: "Node.js Basics", content: "", tags: []string{}, status: "DRAFT"}))
	itSave(t, pool, itMaterial(matSpec{id: "dddd0000-0000-0000-0000-000000000004", group: itGroup2, author: itUserA, title: "Other Group", status: "PUBLISHED", publishedAt: itTimePtr(itNow)}))
}

func itTitles(items []*material.Material) []string {
	out := make([]string, len(items))
	for i, m := range items {
		out[i] = m.Title().String()
	}
	return out
}

func TestRepository_SaveAndFindByID_RoundTripsEveryField(t *testing.T) {
	pool := pgtest.NewDB(t)
	itSeedBase(t, pool)
	published := itNow.Add(time.Hour)
	itSave(t, pool, itMaterial(matSpec{id: itMat1, group: itGroup1, author: itUserA, title: "Graph Algorithms", content: "BFS and DFS", tags: []string{"graphs", "bfs"}, status: "PUBLISHED", pinned: true, publishedAt: &published}))

	got, err := NewRepository(pool).FindByID(context.Background(), itMat1)

	require.NoError(t, err)
	assert.Equal(t, itGroup1, got.GroupID())
	assert.Equal(t, itUserA, got.AuthorID().Value())
	assert.Equal(t, "Graph Algorithms", got.Title().String())
	assert.Equal(t, "BFS and DFS", got.Content().String())
	assert.Equal(t, []string{"graphs", "bfs"}, got.Tags().Values())
	assert.Equal(t, "PUBLISHED", got.Status().String())
	assert.True(t, got.Pinned())
	require.NotNil(t, got.PinnedAt())
	require.NotNil(t, got.PublishedAt())
	assert.True(t, published.Equal(*got.PublishedAt()))
}

func TestRepository_Save_ExistingIDUpdatesTheMutableFields(t *testing.T) {
	pool := pgtest.NewDB(t)
	itSeedBase(t, pool)
	itSave(t, pool, itMaterial(matSpec{id: itMat1, group: itGroup1, author: itUserA, title: "Old", status: "DRAFT"}))
	published := itNow.Add(time.Hour)

	itSave(t, pool, itMaterial(matSpec{id: itMat1, group: itGroup1, author: itUserA, title: "New", status: "PUBLISHED", publishedAt: &published}))

	got, err := NewRepository(pool).FindByID(context.Background(), itMat1)
	require.NoError(t, err)
	assert.Equal(t, "New", got.Title().String())
	assert.Equal(t, "PUBLISHED", got.Status().String())
	var rows int
	require.NoError(t, pool.QueryRow(context.Background(), `SELECT COUNT(*) FROM materials`).Scan(&rows))
	assert.Equal(t, 1, rows)
}

func TestRepository_Save_UnknownGroupReturnsInternalNotRawError(t *testing.T) {
	pool := pgtest.NewDB(t)
	itSeedBase(t, pool)

	err := NewRepository(pool).Save(context.Background(), itMaterial(matSpec{id: itMat1, group: itMissing, author: itUserA, title: "Orphan", status: "DRAFT"}))

	var appErr *apperror.AppError
	require.ErrorAs(t, err, &appErr)
	assert.Equal(t, apperror.KindInternal, appErr.Kind)
}

func TestRepository_FindByIDAndDelete_MissingAndMalformedReportNotFound(t *testing.T) {
	pool := pgtest.NewDB(t)
	itSeedBase(t, pool)
	itSave(t, pool, itMaterial(matSpec{id: itMat1, group: itGroup1, author: itUserA, title: "One", status: "DRAFT"}))
	repo := NewRepository(pool)

	_, err := repo.FindByID(context.Background(), itMissing)
	itRequireNotFound(t, err)
	_, err = repo.FindByID(context.Background(), "not-a-uuid")
	itRequireNotFound(t, err)

	require.NoError(t, repo.Delete(context.Background(), itMat1))
	itRequireNotFound(t, repo.Delete(context.Background(), itMat1))
}

func TestRepository_List_FiltersByGroupStatusViewerAuthorTagsPinnedAndDates(t *testing.T) {
	pool := pgtest.NewDB(t)
	seedLibrary(t, pool)
	repo := NewRepository(pool)
	published := material.NewStatusPublished()
	viewerB, viewerC, authorB := itUserB, itUserC, itUserB
	yes := true
	from := itNow.Add(24 * time.Hour)
	to := itNow

	tests := []struct {
		name      string
		filters   material.ListFilters
		wantTitle []string
		wantTotal int
	}{
		{"the group only, pinned first then newest, drafts without date last", material.ListFilters{}, []string{"Graph Algorithms", "Dynamic Programming", "Node.js Basics"}, 3},
		{"published statuses", material.ListFilters{Statuses: []material.Status{published}}, []string{"Graph Algorithms", "Dynamic Programming"}, 2},
		{"a viewer sees published plus their own drafts", material.ListFilters{ViewerID: &viewerB}, []string{"Graph Algorithms", "Dynamic Programming", "Node.js Basics"}, 3},
		{"a viewer without drafts sees only published", material.ListFilters{ViewerID: &viewerC}, []string{"Graph Algorithms", "Dynamic Programming"}, 2},
		{"by author", material.ListFilters{AuthorID: &authorB}, []string{"Node.js Basics"}, 1},
		{"all tags required", material.ListFilters{Tags: []string{"dp", "graphs"}}, []string{"Dynamic Programming"}, 1},
		{"pinned only", material.ListFilters{Pinned: &yes}, []string{"Graph Algorithms"}, 1},
		{"published from", material.ListFilters{PublishedFrom: &from}, []string{"Dynamic Programming"}, 1},
		{"published to covers the whole boundary day", material.ListFilters{PublishedTo: &to}, []string{"Graph Algorithms"}, 1},
		{"sorted by title", material.ListFilters{SortBy: material.SortByTitle}, []string{"Dynamic Programming", "Graph Algorithms", "Node.js Basics"}, 3},
		{"page two of size two", material.ListFilters{Page: 2, Limit: 2}, []string{"Node.js Basics"}, 3},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.filters.Page == 0 {
				tt.filters.Page = 1
			}
			if tt.filters.Limit == 0 {
				tt.filters.Limit = 10
			}

			got, total, err := repo.List(context.Background(), itGroup1, tt.filters)

			require.NoError(t, err)
			assert.Equal(t, tt.wantTotal, total)
			assert.Equal(t, tt.wantTitle, itTitles(got))
		})
	}
}

func TestRepository_List_FullTextSearch(t *testing.T) {
	pool := pgtest.NewDB(t)
	seedLibrary(t, pool)
	repo := NewRepository(pool)
	search := func(q string, sort material.SortField) []string {
		got, _, err := repo.List(context.Background(), itGroup1, material.ListFilters{SearchQuery: &q, SortBy: sort, Page: 1, Limit: 10})
		require.NoError(t, err)
		return itTitles(got)
	}

	assert.Equal(t, []string{"Graph Algorithms"}, search("graph", material.SortByPublishedAt), "prefix match on the title")
	assert.Equal(t, []string{"Node.js Basics"}, search("node.js", material.SortByPublishedAt), "a dotted name is one token for Postgres")
	assert.Equal(t, []string{"Graph Algorithms", "Dynamic Programming"}, search("algorithms", material.SortByRelevance), "a title hit outranks a content hit")
	assert.Equal(t, []string{"Dynamic Programming", "Graph Algorithms"}, search("algorithms", material.SortByPublishedAt), "searching drops the pinned boost: newest first")
	assert.Equal(t, []string{"Graph Algorithms", "Dynamic Programming", "Node.js Basics"}, search("", material.SortByRelevance), "no query: the regular order")
	assert.Empty(t, search("zzzz", material.SortByPublishedAt))
}

func TestRepository_DistinctTagsAndAuthorIDs_ScopedToTheGroupAndStatuses(t *testing.T) {
	pool := pgtest.NewDB(t)
	seedLibrary(t, pool)
	repo := NewRepository(pool)

	tags, err := repo.DistinctTags(context.Background(), itGroup1, nil, 10)
	require.NoError(t, err)
	assert.Equal(t, []string{"dp", "graphs"}, tags)

	publishedOnly, err := repo.DistinctTags(context.Background(), itGroup1, []material.Status{material.NewStatusPublished()}, 1)
	require.NoError(t, err)
	assert.Equal(t, []string{"dp", "graphs"}, publishedOnly, "sample of the most recent material only")

	authors, err := repo.DistinctAuthorIDs(context.Background(), itGroup1, nil, 10)
	require.NoError(t, err)
	assert.ElementsMatch(t, []string{itUserA, itUserB}, authors)

	draftAuthors, err := repo.DistinctAuthorIDs(context.Background(), itGroup1, []material.Status{material.NewStatusDraft()}, 10)
	require.NoError(t, err)
	assert.Equal(t, []string{itUserB}, draftAuthors)

	none, err := repo.DistinctTags(context.Background(), itMissing, nil, 10)
	require.NoError(t, err)
	assert.Empty(t, none)
}

func TestMaterialProviders_AuthorsGroupsMembershipAndDashboard(t *testing.T) {
	pool := pgtest.NewDB(t)
	seedLibrary(t, pool)
	pgtest.SeedGroupMember(t, pool, "77770000-0000-0000-0000-0000000000a1", itGroup1, itUserA, "LEAD")
	pgtest.SeedGroupMember(t, pool, "77770000-0000-0000-0000-0000000000a2", itGroup1, itUserB, "MEMBER")
	ctx := context.Background()

	id, found, err := NewAuthorIDProvider(pool).FindIDByNickname(ctx, "ANA")
	require.NoError(t, err)
	assert.True(t, found)
	assert.Equal(t, itUserA, id)
	_, found, err = NewAuthorIDProvider(pool).FindIDByNickname(ctx, "nobody")
	require.NoError(t, err)
	assert.False(t, found)

	displays, err := NewAuthorProvider(pool).GetDisplays(ctx, []string{itUserA, itMissing})
	require.NoError(t, err)
	require.Len(t, displays, 1)
	assert.Equal(t, "ana", displays[itUserA].Nickname)
	empty, err := NewAuthorProvider(pool).GetDisplays(ctx, nil)
	require.NoError(t, err)
	assert.Empty(t, empty)

	groups := NewGroupProvider(pool)
	exists, err := groups.Exists(ctx, itGroup1)
	require.NoError(t, err)
	assert.True(t, exists)
	exists, err = groups.Exists(ctx, "not-a-uuid")
	require.NoError(t, err)
	assert.False(t, exists)
	visibility, ok, err := groups.FindVisibility(ctx, itGroup1)
	require.NoError(t, err)
	assert.True(t, ok)
	assert.Equal(t, appMaterial.GroupVisibility("VISIBLE"), visibility)
	_, ok, err = groups.FindVisibility(ctx, itMissing)
	require.NoError(t, err)
	assert.False(t, ok)

	members := NewGroupMemberProvider(pool)
	lead, err := members.IsLeadOfGroup(ctx, itUserA, itGroup1)
	require.NoError(t, err)
	assert.True(t, lead)
	lead, err = members.IsLeadOfGroup(ctx, itUserB, itGroup1)
	require.NoError(t, err)
	assert.False(t, lead)
	lead, err = members.IsLeadOfGroup(ctx, itUserC, itGroup1)
	require.NoError(t, err)
	assert.False(t, lead)
	member, err := members.IsMemberOfGroup(ctx, itUserB, itGroup1)
	require.NoError(t, err)
	assert.True(t, member)
	member, err = members.IsMemberOfGroup(ctx, itUserC, "not-a-uuid")
	require.NoError(t, err)
	assert.False(t, member)

	recent := time.Now().Add(-24 * time.Hour)
	itSave(t, pool, itMaterial(matSpec{id: "dddd0000-0000-0000-0000-000000000009", group: itGroup1, author: itUserA, title: "Fresh", status: "PUBLISHED", publishedAt: &recent}))
	count, err := NewDashboardProvider(pool).GetRecentMaterialsCount(ctx, itUserB, 7)
	require.NoError(t, err)
	assert.Equal(t, 1, count, "only published materials of the user's groups inside the window")
	count, err = NewDashboardProvider(pool).GetRecentMaterialsCount(ctx, itUserC, 7)
	require.NoError(t, err)
	assert.Equal(t, 0, count)
}
