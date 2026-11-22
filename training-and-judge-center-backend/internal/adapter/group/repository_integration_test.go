//go:build integration

package group

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	domainGroup "github.com/training-judge-center/backend/internal/domain/group"
	"github.com/training-judge-center/backend/internal/domain/shared"
	"github.com/training-judge-center/backend/internal/testutil/pgtest"
	"github.com/training-judge-center/backend/pkg/apperror"
)

func TestMain(m *testing.M) {
	os.Exit(pgtest.Main(m))
}

var itNow = time.Date(2026, 8, 8, 12, 0, 0, 0, time.UTC)

const (
	itUserA    = "11111111-1111-1111-1111-111111111111"
	itUserB    = "22222222-2222-2222-2222-222222222222"
	itUserC    = "33333333-3333-3333-3333-333333333333"
	itGroup1   = "bbbbbbbb-0000-0000-0000-000000000001"
	itGroup2   = "bbbbbbbb-0000-0000-0000-000000000002"
	itGroup3   = "bbbbbbbb-0000-0000-0000-000000000003"
	itDefault  = "00000000-0000-0000-0000-000000000002"
	itMissing  = "99999999-9999-9999-9999-999999999999"
	itMemberID = "77770000-0000-0000-0000-0000000000"
)

func itSeedUsers(t *testing.T, pool *pgxpool.Pool) {
	t.Helper()
	pgtest.SeedUser(t, pool, itUserA, "ana")
	pgtest.SeedUser(t, pool, itUserB, "beto")
	pgtest.SeedUser(t, pool, itUserC, "carla")
}

func itGroup(t *testing.T, id, name string, visibility domainGroup.Visibility, policy domainGroup.JoinPolicy, createdAt time.Time) *domainGroup.Group {
	t.Helper()
	groupName, err := domainGroup.NewGroupName(name)
	require.NoError(t, err)
	desc := "About " + name
	g, err := domainGroup.NewGroup(id, groupName, &desc, visibility, policy, shared.RestoreUserID(itUserA), createdAt)
	require.NoError(t, err)
	return g
}

func itSave(t *testing.T, pool *pgxpool.Pool, g *domainGroup.Group) {
	t.Helper()
	require.NoError(t, NewRepository(pool).Save(context.Background(), g))
}

func itRequireCode(t *testing.T, err error, code string) {
	t.Helper()
	var appErr *apperror.AppError
	require.ErrorAs(t, err, &appErr)
	assert.Equal(t, code, appErr.Code)
}

func itGroupNames(groups []*domainGroup.Group) []string {
	out := make([]string, len(groups))
	for i, g := range groups {
		out[i] = g.Name().Value()
	}
	return out
}

func TestRepository_Save_RoundTripsEveryField(t *testing.T) {
	pool := pgtest.NewDB(t)
	itSeedUsers(t, pool)
	itSave(t, pool, itGroup(t, itGroup1, "Algorithms", domainGroup.VisibilityNotVisible, domainGroup.JoinPolicyInvite, itNow))

	got, err := NewRepository(pool).FindByID(context.Background(), itGroup1)

	require.NoError(t, err)
	assert.Equal(t, "Algorithms", got.Name().Value())
	require.NotNil(t, got.Description())
	assert.Equal(t, "About Algorithms", *got.Description())
	assert.Equal(t, "NOT_VISIBLE", got.Visibility().String())
	assert.Equal(t, "INVITE", got.JoinPolicy().String())
	assert.False(t, got.IsDefault())
	assert.Equal(t, itUserA, got.CreatedBy().Value())
	assert.True(t, itNow.Equal(got.CreatedAt()))
}

func TestRepository_Save_DuplicateNameIsCaseInsensitiveConflict(t *testing.T) {
	pool := pgtest.NewDB(t)
	itSeedUsers(t, pool)
	itSave(t, pool, itGroup(t, itGroup1, "Algorithms", domainGroup.VisibilityVisible, domainGroup.JoinPolicyOpen, itNow))

	err := NewRepository(pool).Save(context.Background(), itGroup(t, itGroup2, "ALGORITHMS", domainGroup.VisibilityVisible, domainGroup.JoinPolicyOpen, itNow))

	itRequireCode(t, err, domainGroup.ErrCodeNameAlreadyExists)
}

func TestRepository_FindByID_MissingAndMalformedReturnNotFound(t *testing.T) {
	pool := pgtest.NewDB(t)
	repo := NewRepository(pool)

	_, err := repo.FindByID(context.Background(), itMissing)
	itRequireCode(t, err, domainGroup.ErrCodeGroupNotFound)

	_, err = repo.FindByID(context.Background(), "not-a-uuid")
	itRequireCode(t, err, domainGroup.ErrCodeGroupNotFound)
}

func TestRepository_ExistsByName_IsCaseInsensitive(t *testing.T) {
	pool := pgtest.NewDB(t)
	itSeedUsers(t, pool)
	itSave(t, pool, itGroup(t, itGroup1, "Algorithms", domainGroup.VisibilityVisible, domainGroup.JoinPolicyOpen, itNow))
	repo := NewRepository(pool)
	same, err := domainGroup.NewGroupName("aLgOrItHmS")
	require.NoError(t, err)
	other, err := domainGroup.NewGroupName("Data Structures")
	require.NoError(t, err)

	exists, err := repo.ExistsByName(context.Background(), same)
	require.NoError(t, err)
	assert.True(t, exists)

	exists, err = repo.ExistsByName(context.Background(), other)
	require.NoError(t, err)
	assert.False(t, exists)
}

func TestRepository_FindDefault_ReturnsTheSeededGlobalGroup(t *testing.T) {
	pool := pgtest.NewDB(t)

	got, err := NewRepository(pool).FindDefault(context.Background())

	require.NoError(t, err)
	assert.Equal(t, itDefault, got.ID())
	assert.Equal(t, "Global", got.Name().Value())
	assert.True(t, got.IsDefault())
}

func TestRepository_Update_PersistsChangesAndRejectsATakenName(t *testing.T) {
	pool := pgtest.NewDB(t)
	itSeedUsers(t, pool)
	first := itGroup(t, itGroup1, "Algorithms", domainGroup.VisibilityVisible, domainGroup.JoinPolicyOpen, itNow)
	itSave(t, pool, first)
	itSave(t, pool, itGroup(t, itGroup2, "Data Structures", domainGroup.VisibilityVisible, domainGroup.JoinPolicyOpen, itNow))
	repo := NewRepository(pool)

	loaded, err := repo.FindByID(context.Background(), itGroup1)
	require.NoError(t, err)
	newName, err := domainGroup.NewGroupName("Graph Theory")
	require.NoError(t, err)
	loaded.UpdateMetadata(&newName, nil, itNow.Add(time.Hour))
	require.NoError(t, repo.Update(context.Background(), loaded))

	got, err := repo.FindByID(context.Background(), itGroup1)
	require.NoError(t, err)
	assert.Equal(t, "Graph Theory", got.Name().Value())

	taken, err := domainGroup.NewGroupName("Data Structures")
	require.NoError(t, err)
	got.UpdateMetadata(&taken, nil, itNow.Add(2*time.Hour))
	itRequireCode(t, repo.Update(context.Background(), got), domainGroup.ErrCodeNameAlreadyExists)
}

func TestRepository_Delete_RemovesTheGroupAndItsMembers(t *testing.T) {
	pool := pgtest.NewDB(t)
	itSeedUsers(t, pool)
	itSave(t, pool, itGroup(t, itGroup1, "Algorithms", domainGroup.VisibilityVisible, domainGroup.JoinPolicyOpen, itNow))
	pgtest.SeedGroupMember(t, pool, itMemberID+"01", itGroup1, itUserB, "MEMBER")
	repo := NewRepository(pool)

	require.NoError(t, repo.Delete(context.Background(), itGroup1))

	_, err := repo.FindByID(context.Background(), itGroup1)
	itRequireCode(t, err, domainGroup.ErrCodeGroupNotFound)
	var members int
	require.NoError(t, pool.QueryRow(context.Background(), `SELECT COUNT(*) FROM group_members WHERE group_id = $1`, itGroup1).Scan(&members))
	assert.Equal(t, 0, members)
	itRequireCode(t, repo.Delete(context.Background(), itGroup1), domainGroup.ErrCodeGroupNotFound)
}

func seedGroupCatalog(t *testing.T, pool *pgxpool.Pool) {
	t.Helper()
	itSeedUsers(t, pool)
	itSave(t, pool, itGroup(t, itGroup1, "Alpha Open", domainGroup.VisibilityVisible, domainGroup.JoinPolicyOpen, itNow))
	itSave(t, pool, itGroup(t, itGroup2, "Beta Hidden", domainGroup.VisibilityNotVisible, domainGroup.JoinPolicyInvite, itNow.Add(time.Hour)))
	itSave(t, pool, itGroup(t, itGroup3, "Gamma Request", domainGroup.VisibilityVisible, domainGroup.JoinPolicyRequest, itNow.Add(2*time.Hour)))
	pgtest.SeedGroupMember(t, pool, itMemberID+"01", itGroup2, itUserB, "LEAD")
	pgtest.SeedGroupMember(t, pool, itMemberID+"02", itGroup2, itUserC, "MEMBER")
	pgtest.SeedGroupMember(t, pool, itMemberID+"03", itGroup1, itUserB, "MEMBER")
}

func TestRepository_List_VisibilityDependsOnTheViewer(t *testing.T) {
	pool := pgtest.NewDB(t)
	seedGroupCatalog(t, pool)
	repo := NewRepository(pool)

	tests := []struct {
		name      string
		viewer    string
		admin     bool
		wantNames []string
	}{
		{"an outsider does not see the hidden group", itUserA, false, []string{"Alpha Open", "Gamma Request"}},
		{"a member sees the hidden group they belong to", itUserB, false, []string{"Alpha Open", "Beta Hidden", "Gamma Request"}},
		{"an admin sees every group", itUserA, true, []string{"Alpha Open", "Beta Hidden", "Gamma Request"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, total, err := repo.List(context.Background(), domainGroup.ListFilters{
				ViewerID: shared.RestoreUserID(tt.viewer), ViewerIsAdmin: tt.admin, ExcludeDefault: true, Page: 1, Limit: 10,
			})

			require.NoError(t, err)
			assert.Equal(t, len(tt.wantNames), total)
			assert.Equal(t, tt.wantNames, itGroupNames(got))
		})
	}
}

func TestRepository_List_FiltersSortsAndPaginates(t *testing.T) {
	pool := pgtest.NewDB(t)
	seedGroupCatalog(t, pool)
	repo := NewRepository(pool)
	viewerB := shared.RestoreUserID(itUserB)
	lead := domainGroup.RestoreMemberRole("LEAD")
	hidden := domainGroup.VisibilityNotVisible
	request := domainGroup.JoinPolicyRequest

	tests := []struct {
		name      string
		filters   domainGroup.ListFilters
		wantNames []string
		wantTotal int
	}{
		{"default sort by name descending", domainGroup.ListFilters{Order: domainGroup.OrderDesc}, []string{"Gamma Request", "Beta Hidden", "Alpha Open"}, 3},
		{"only groups I belong to", domainGroup.ListFilters{OnlyMyGroups: &domainGroup.MembershipFilter{}}, []string{"Alpha Open", "Beta Hidden"}, 2},
		{"only groups where I am lead", domainGroup.ListFilters{OnlyMyGroups: &domainGroup.MembershipFilter{RoleFilter: &lead}}, []string{"Beta Hidden"}, 1},
		{"by visibility", domainGroup.ListFilters{Visibility: &hidden}, []string{"Beta Hidden"}, 1},
		{"by join policy", domainGroup.ListFilters{JoinPolicy: &request}, []string{"Gamma Request"}, 1},
		{"search by name", domainGroup.ListFilters{Search: "GAMMA"}, []string{"Gamma Request"}, 1},
		{"search also looks in the description", domainGroup.ListFilters{Search: "about beta"}, []string{"Beta Hidden"}, 1},
		{"newest created first", domainGroup.ListFilters{SortBy: domainGroup.SortByCreatedAt, Order: domainGroup.OrderDesc}, []string{"Gamma Request", "Beta Hidden", "Alpha Open"}, 3},
		{"most members first", domainGroup.ListFilters{SortBy: domainGroup.SortByMemberCount, Order: domainGroup.OrderDesc}, []string{"Beta Hidden", "Alpha Open", "Gamma Request"}, 3},
		{"sorted by when I joined", domainGroup.ListFilters{SortBy: domainGroup.SortByJoinedAt, Order: domainGroup.OrderAsc, OnlyMyGroups: &domainGroup.MembershipFilter{}}, []string{"Beta Hidden", "Alpha Open"}, 2},
		{"page two of size two", domainGroup.ListFilters{Page: 2, Limit: 2}, []string{"Gamma Request"}, 3},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.filters.ViewerID = viewerB
			tt.filters.ExcludeDefault = true
			if tt.filters.Page == 0 {
				tt.filters.Page = 1
			}
			if tt.filters.Limit == 0 {
				tt.filters.Limit = 10
			}

			got, total, err := repo.List(context.Background(), tt.filters)

			require.NoError(t, err)
			assert.Equal(t, tt.wantTotal, total)
			assert.Equal(t, tt.wantNames, itGroupNames(got))
		})
	}
}

func TestRepository_List_IncludesTheDefaultGroupUnlessExcluded(t *testing.T) {
	pool := pgtest.NewDB(t)
	seedGroupCatalog(t, pool)
	repo := NewRepository(pool)
	filters := domainGroup.ListFilters{ViewerID: shared.RestoreUserID(itUserA), ViewerIsAdmin: true, Page: 1, Limit: 10}

	with, total, err := repo.List(context.Background(), filters)
	require.NoError(t, err)
	assert.Equal(t, 4, total)
	assert.Contains(t, itGroupNames(with), "Global")

	filters.ExcludeDefault = true
	without, total, err := repo.List(context.Background(), filters)
	require.NoError(t, err)
	assert.Equal(t, 3, total)
	assert.NotContains(t, itGroupNames(without), "Global")
}
