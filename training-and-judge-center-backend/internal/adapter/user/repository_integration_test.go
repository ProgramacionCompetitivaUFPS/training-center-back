//go:build integration

package user

import (
	"context"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/training-judge-center/backend/internal/domain/shared"
	domainUser "github.com/training-judge-center/backend/internal/domain/user"
	"github.com/training-judge-center/backend/internal/testutil/pgtest"
	"github.com/training-judge-center/backend/pkg/apperror"
)

var integrationNow = time.Date(2026, 8, 8, 12, 0, 0, 0, time.UTC)

const (
	userA = "11111111-1111-1111-1111-111111111111"
	userB = "22222222-2222-2222-2222-222222222222"
	userC = "33333333-3333-3333-3333-333333333333"
)

type userSpec struct {
	id          string
	email       string
	nickname    string
	name        string
	country     string
	city        string
	institution string
}

func newTestUser(t *testing.T, s userSpec) *domainUser.User {
	t.Helper()
	if s.name == "" {
		s.name = "Test User"
	}
	if s.country == "" {
		s.country = "Colombia"
	}
	if s.city == "" {
		s.city = "Bogota"
	}
	if s.institution == "" {
		s.institution = "UNAL"
	}
	email, err := domainUser.NewEmail(s.email)
	require.NoError(t, err)
	nickname, err := domainUser.NewNickname(s.nickname)
	require.NoError(t, err)
	u, err := domainUser.NewUser(s.id, integrationNow, email, domainUser.RestorePassword("hash"), s.name, nickname, s.country, s.city, s.institution)
	require.NoError(t, err)
	return u
}

func seedUser(t *testing.T, pool *pgxpool.Pool, s userSpec) *domainUser.User {
	t.Helper()
	u := newTestUser(t, s)
	require.NoError(t, NewRepository(pool).Save(context.Background(), u))
	return u
}

func seedDirectory(t *testing.T, pool *pgxpool.Pool) {
	t.Helper()
	seedUser(t, pool, userSpec{id: userA, email: "ana@example.com", nickname: "ana", name: "Ana Perez", country: "Colombia", city: "Cali", institution: "Univalle"})
	seedUser(t, pool, userSpec{id: userB, email: "beto@example.com", nickname: "beto", name: "Beto Ruiz", country: "Peru", city: "Lima", institution: "UNI"})
	seedUser(t, pool, userSpec{id: userC, email: "carla@example.com", nickname: "carla", name: "Carla 100%", country: "Colombia", city: "Bogota", institution: "UNAL"})
}

func deactivateRow(t *testing.T, pool *pgxpool.Pool, id string) {
	t.Helper()
	_, err := pool.Exec(context.Background(), `UPDATE users SET status = 'DEACTIVATED', email = NULL, deactivated_at = NOW() WHERE id = $1`, id)
	require.NoError(t, err)
}

func filterWith(t *testing.T, mutate func(f *domainUser.UserFilter)) domainUser.UserFilter {
	t.Helper()
	f, err := domainUser.NewUserFilter(nil, nil, "", "", "", domainUser.SearchByAll, "", domainUser.SortByName, domainUser.SortOrderAsc, 1, 10)
	require.NoError(t, err)
	active := domainUser.StatusActive
	f.Status = &active
	mutate(&f)
	return f
}

func nicknames(users []*domainUser.User) []string {
	out := make([]string, len(users))
	for i, u := range users {
		out[i] = u.Nickname().String()
	}
	return out
}

func requireConflict(t *testing.T, err error, code string) {
	t.Helper()
	var appErr *apperror.AppError
	require.ErrorAs(t, err, &appErr)
	assert.Equal(t, code, appErr.Code)
}

func TestRepository_Save_RoundTripsEveryField(t *testing.T) {
	pool := pgtest.NewDB(t)
	repo := NewRepository(pool)
	u := newTestUser(t, userSpec{id: userA, email: "ana@example.com", nickname: "ana", name: "Ana Perez", country: "Colombia", city: "Cali", institution: "Univalle"})

	require.NoError(t, repo.Save(context.Background(), u))

	got, err := repo.FindByID(context.Background(), userA)
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, "ana@example.com", got.Email().String())
	assert.Equal(t, "ana", got.Nickname().String())
	assert.Equal(t, "Ana Perez", got.Name())
	assert.Equal(t, "Colombia", got.Country())
	assert.Equal(t, "Cali", got.City())
	assert.Equal(t, "Univalle", got.Institution())
	assert.Equal(t, "CONTESTANT", got.Role().String())
	assert.Equal(t, "ACTIVE", got.Status().String())
	assert.True(t, integrationNow.Equal(got.CreatedAt()))
}

func TestRepository_Save_DuplicateEmailReturnsEmailConflict(t *testing.T) {
	pool := pgtest.NewDB(t)
	seedUser(t, pool, userSpec{id: userA, email: "ana@example.com", nickname: "ana"})

	err := NewRepository(pool).Save(context.Background(), newTestUser(t, userSpec{id: userB, email: "ana@example.com", nickname: "other"}))

	requireConflict(t, err, domainUser.ErrCodeEmailConflict)
}

func TestRepository_Save_DuplicateNicknameReturnsNicknameConflict(t *testing.T) {
	pool := pgtest.NewDB(t)
	seedUser(t, pool, userSpec{id: userA, email: "ana@example.com", nickname: "ana"})

	err := NewRepository(pool).Save(context.Background(), newTestUser(t, userSpec{id: userB, email: "other@example.com", nickname: "ana"}))

	requireConflict(t, err, domainUser.ErrCodeNicknameConflict)
}

func TestRepository_Update_PersistsChanges(t *testing.T) {
	pool := pgtest.NewDB(t)
	repo := NewRepository(pool)
	a := seedUser(t, pool, userSpec{id: userA, email: "ana@example.com", nickname: "ana"})

	newName := "Ana Maria"
	require.NoError(t, a.Update(&newName, nil, nil, nil, nil, integrationNow.Add(time.Hour)))
	require.NoError(t, repo.Update(context.Background(), a))

	got, err := repo.FindByID(context.Background(), userA)
	require.NoError(t, err)
	assert.Equal(t, "Ana Maria", got.Name())
	require.NotNil(t, got.UpdatedAt())
	assert.True(t, integrationNow.Add(time.Hour).Equal(*got.UpdatedAt()))
}

func TestRepository_Update_NicknameTakenByAnotherReturnsConflict(t *testing.T) {
	pool := pgtest.NewDB(t)
	a := seedUser(t, pool, userSpec{id: userA, email: "ana@example.com", nickname: "ana"})
	seedUser(t, pool, userSpec{id: userB, email: "beto@example.com", nickname: "beto"})

	taken, err := domainUser.NewNickname("beto")
	require.NoError(t, err)
	require.NoError(t, a.Update(nil, &taken, nil, nil, nil, integrationNow.Add(time.Hour)))

	requireConflict(t, NewRepository(pool).Update(context.Background(), a), domainUser.ErrCodeNicknameConflict)
}

func TestRepository_Update_EmailTakenByAnotherReturnsConflict(t *testing.T) {
	pool := pgtest.NewDB(t)
	a := seedUser(t, pool, userSpec{id: userA, email: "ana@example.com", nickname: "ana"})
	seedUser(t, pool, userSpec{id: userB, email: "beto@example.com", nickname: "beto"})

	taken, err := domainUser.NewEmail("beto@example.com")
	require.NoError(t, err)
	require.NoError(t, a.UpdateEmail(taken, integrationNow.Add(time.Hour)))

	requireConflict(t, NewRepository(pool).Update(context.Background(), a), domainUser.ErrCodeEmailConflict)
}

func TestRepository_Find_ReturnsNilWhenMissingOrMalformed(t *testing.T) {
	pool := pgtest.NewDB(t)
	repo := NewRepository(pool)
	ctx := context.Background()

	byID, err := repo.FindByID(ctx, userC)
	require.NoError(t, err)
	assert.Nil(t, byID)

	email, err := domainUser.NewEmail("nobody@example.com")
	require.NoError(t, err)
	byEmail, err := repo.FindByEmail(ctx, email)
	require.NoError(t, err)
	assert.Nil(t, byEmail)

	nick, err := domainUser.NewNickname("nobody")
	require.NoError(t, err)
	byNick, err := repo.FindByNickname(ctx, nick)
	require.NoError(t, err)
	assert.Nil(t, byNick)
}

func TestRepository_FindByID_MalformedUUIDReturnsNilNotAnError(t *testing.T) {
	pool := pgtest.NewDB(t)

	got, err := NewRepository(pool).FindByID(context.Background(), "not-a-uuid")

	require.NoError(t, err)
	assert.Nil(t, got)
}

func TestRepository_FindByEmailAndNickname_ReturnTheUser(t *testing.T) {
	pool := pgtest.NewDB(t)
	repo := NewRepository(pool)
	seedUser(t, pool, userSpec{id: userA, email: "ana@example.com", nickname: "ana"})

	email, err := domainUser.NewEmail("ana@example.com")
	require.NoError(t, err)
	byEmail, err := repo.FindByEmail(context.Background(), email)
	require.NoError(t, err)
	require.NotNil(t, byEmail)
	assert.Equal(t, userA, byEmail.ID())

	nick, err := domainUser.NewNickname("ana")
	require.NoError(t, err)
	byNick, err := repo.FindByNickname(context.Background(), nick)
	require.NoError(t, err)
	require.NotNil(t, byNick)
	assert.Equal(t, userA, byNick.ID())
}

func TestRepository_FindAll_FiltersSearchesAndPaginates(t *testing.T) {
	pool := pgtest.NewDB(t)
	seedDirectory(t, pool)
	repo := NewRepository(pool)

	tests := []struct {
		name      string
		mutate    func(f *domainUser.UserFilter)
		wantNicks []string
		wantTotal int
	}{
		{"no filter sorted by name", func(f *domainUser.UserFilter) {}, []string{"ana", "beto", "carla"}, 3},
		{"country is case-insensitive", func(f *domainUser.UserFilter) { f.Country = "colombia" }, []string{"ana", "carla"}, 2},
		{"city", func(f *domainUser.UserFilter) { f.City = "LIMA" }, []string{"beto"}, 1},
		{"institution", func(f *domainUser.UserFilter) { f.Institution = "unal" }, []string{"carla"}, 1},
		{"search by name", func(f *domainUser.UserFilter) { f.SearchField = domainUser.SearchByName; f.SearchTerm = "ruiz" }, []string{"beto"}, 1},
		{"search by nickname", func(f *domainUser.UserFilter) { f.SearchField = domainUser.SearchByNickname; f.SearchTerm = "AN" }, []string{"ana"}, 1},
		{"search by email", func(f *domainUser.UserFilter) { f.SearchField = domainUser.SearchByEmail; f.SearchTerm = "carla@" }, []string{"carla"}, 1},
		{"search by institution", func(f *domainUser.UserFilter) { f.SearchField = domainUser.SearchByInstitution; f.SearchTerm = "uni" }, []string{"beto", "ana"}, 2},
		{"search all matches institution", func(f *domainUser.UserFilter) { f.SearchTerm = "univalle" }, []string{"ana"}, 1},
		{"percent is literal not wildcard", func(f *domainUser.UserFilter) { f.SearchField = domainUser.SearchByName; f.SearchTerm = "%" }, []string{"carla"}, 1},
		{"descending order", func(f *domainUser.UserFilter) { f.Order = domainUser.SortOrderDesc }, []string{"carla", "beto", "ana"}, 3},
		{"second page of size two", func(f *domainUser.UserFilter) { f.Page = 2; f.Limit = 2 }, []string{"carla"}, 3},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			users, total, err := repo.FindAll(context.Background(), filterWith(t, tt.mutate))

			require.NoError(t, err)
			assert.Equal(t, tt.wantTotal, total)
			assert.ElementsMatch(t, tt.wantNicks, nicknames(users))
		})
	}
}

func TestRepository_FindAll_FiltersByRoleAndStatus(t *testing.T) {
	pool := pgtest.NewDB(t)
	seedDirectory(t, pool)
	_, err := pool.Exec(context.Background(), `UPDATE users SET role = 'ADMIN' WHERE id = $1`, userB)
	require.NoError(t, err)
	deactivateRow(t, pool, userC)
	repo := NewRepository(pool)

	users, total, err := repo.FindAll(context.Background(), filterWith(t, func(f *domainUser.UserFilter) { f.Roles = []shared.Role{shared.RoleAdmin} }))
	require.NoError(t, err)
	assert.Equal(t, 1, total)
	assert.Equal(t, []string{"beto"}, nicknames(users))

	deactivated := domainUser.RestoreStatus("DEACTIVATED")
	users, total, err = repo.FindAll(context.Background(), filterWith(t, func(f *domainUser.UserFilter) { f.Status = &deactivated }))
	require.NoError(t, err)
	assert.Equal(t, 2, total)
	assert.ElementsMatch(t, []string{"carla", "system"}, nicknames(users))
}

func TestRepository_SearchActive_MatchesNameOrNicknameOnlyAndSkipsInactive(t *testing.T) {
	pool := pgtest.NewDB(t)
	seedDirectory(t, pool)
	deactivateRow(t, pool, userC)
	repo := NewRepository(pool)

	byName, err := repo.SearchActive(context.Background(), "perez", 10)
	require.NoError(t, err)
	assert.Equal(t, []string{"ana"}, nicknames(byName))

	byEmailOnly, err := repo.SearchActive(context.Background(), "example.com", 10)
	require.NoError(t, err)
	assert.Empty(t, byEmailOnly)

	inactive, err := repo.SearchActive(context.Background(), "carla", 10)
	require.NoError(t, err)
	assert.Empty(t, inactive)

	capped, err := repo.SearchActive(context.Background(), "e", 1)
	require.NoError(t, err)
	assert.Len(t, capped, 1)
}

func TestRepository_FindFilterOptions_ReturnsDistinctSortedNonEmptyValues(t *testing.T) {
	pool := pgtest.NewDB(t)
	seedDirectory(t, pool)

	got, err := NewRepository(pool).FindFilterOptions(context.Background())

	require.NoError(t, err)
	assert.Equal(t, []string{"Colombia", "N/A", "Peru"}, got.Countries)
	assert.Equal(t, []string{"Bogota", "Cali", "Lima", "N/A"}, got.Cities)
	assert.Equal(t, []string{"N/A", "UNAL", "UNI", "Univalle"}, got.Institutions)
}
