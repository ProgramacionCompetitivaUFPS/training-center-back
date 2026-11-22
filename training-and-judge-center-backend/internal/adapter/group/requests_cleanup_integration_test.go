//go:build integration

package group

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	domainGroup "github.com/training-judge-center/backend/internal/domain/group"
	"github.com/training-judge-center/backend/internal/domain/shared"
	"github.com/training-judge-center/backend/internal/testutil/pgtest"
)

const (
	itRequest1 = "aaaa1111-0000-0000-0000-000000000001"
	itRequest2 = "aaaa1111-0000-0000-0000-000000000002"
	itContest1 = "cccccccc-0000-0000-0000-000000000001"
	itContest2 = "cccccccc-0000-0000-0000-000000000002"
	itTeam1    = "eeeeeeee-0000-0000-0000-000000000001"
	itProblem1 = "aaaaaaaa-0000-0000-0000-000000000001"
)

func itJoinRequest(t *testing.T, id, userID string, message *string, at time.Time) *domainGroup.JoinRequest {
	t.Helper()
	req, err := domainGroup.NewJoinRequest(id, itGroup1, shared.RestoreUserID(userID), message, at)
	require.NoError(t, err)
	return req
}

func TestJoinRequestRepository_SaveAndFind_RoundTrips(t *testing.T) {
	pool := pgtest.NewDB(t)
	itSeedGroupWithOwner(t, pool)
	repo := NewJoinRequestRepository(pool)
	message := "let me in"
	require.NoError(t, repo.Save(context.Background(), itJoinRequest(t, itRequest1, itUserB, &message, itNow)))

	got, err := repo.FindByID(context.Background(), itRequest1)

	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, itUserB, got.RequesterUserID().Value())
	require.NotNil(t, got.Message())
	assert.Equal(t, "let me in", *got.Message())
	assert.Equal(t, "PENDING", got.Status().String())

	missing, err := repo.FindByID(context.Background(), itMissing)
	require.NoError(t, err)
	assert.Nil(t, missing)
	malformed, err := repo.FindByID(context.Background(), "not-a-uuid")
	require.NoError(t, err)
	assert.Nil(t, malformed)
}

func TestJoinRequestRepository_Save_SecondPendingRequestIsRefused(t *testing.T) {
	pool := pgtest.NewDB(t)
	itSeedGroupWithOwner(t, pool)
	repo := NewJoinRequestRepository(pool)
	require.NoError(t, repo.Save(context.Background(), itJoinRequest(t, itRequest1, itUserB, nil, itNow)))

	err := repo.Save(context.Background(), itJoinRequest(t, itRequest2, itUserB, nil, itNow.Add(time.Hour)))

	itRequireCode(t, err, domainGroup.ErrCodeRequestAlreadyPending)
}

func TestJoinRequestRepository_Save_SameIDUpdatesTheStatus(t *testing.T) {
	pool := pgtest.NewDB(t)
	itSeedGroupWithOwner(t, pool)
	repo := NewJoinRequestRepository(pool)
	req := itJoinRequest(t, itRequest1, itUserB, nil, itNow)
	require.NoError(t, repo.Save(context.Background(), req))

	approved := domainGroup.RestoreJoinRequest(itRequest1, itGroup1, shared.RestoreUserID(itUserB), nil, domainGroup.JoinRequestStatusApproved, itNow)
	require.NoError(t, repo.Save(context.Background(), approved))

	got, err := repo.FindByID(context.Background(), itRequest1)
	require.NoError(t, err)
	assert.Equal(t, "APPROVED", got.Status().String())
}

func TestJoinRequestRepository_FindByGroupAndUser_PrefersThePendingOne(t *testing.T) {
	pool := pgtest.NewDB(t)
	itSeedGroupWithOwner(t, pool)
	repo := NewJoinRequestRepository(pool)
	old := domainGroup.RestoreJoinRequest(itRequest1, itGroup1, shared.RestoreUserID(itUserB), nil, domainGroup.JoinRequestStatusRejected, itNow.Add(2*time.Hour))
	require.NoError(t, repo.Save(context.Background(), old))
	require.NoError(t, repo.Save(context.Background(), itJoinRequest(t, itRequest2, itUserB, nil, itNow)))

	got, err := repo.FindByGroupAndUser(context.Background(), itGroup1, shared.RestoreUserID(itUserB))
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, itRequest2, got.ID(), "the pending request wins even though the rejected one is newer")

	none, err := repo.FindByGroupAndUser(context.Background(), itGroup1, shared.RestoreUserID(itUserC))
	require.NoError(t, err)
	assert.Nil(t, none)
}

func TestJoinRequestRepository_FindByGroup_FiltersAndPaginatesNewestFirst(t *testing.T) {
	pool := pgtest.NewDB(t)
	itSeedGroupWithOwner(t, pool)
	repo := NewJoinRequestRepository(pool)
	require.NoError(t, repo.Save(context.Background(), domainGroup.RestoreJoinRequest(itRequest1, itGroup1, shared.RestoreUserID(itUserA), nil, domainGroup.JoinRequestStatusRejected, itNow)))
	require.NoError(t, repo.Save(context.Background(), itJoinRequest(t, itRequest2, itUserB, nil, itNow.Add(time.Hour))))
	require.NoError(t, repo.Save(context.Background(), itJoinRequest(t, "aaaa1111-0000-0000-0000-000000000003", itUserC, nil, itNow.Add(2*time.Hour))))

	all, total, err := repo.FindByGroup(context.Background(), itGroup1, domainGroup.JoinRequestFilters{})
	require.NoError(t, err)
	assert.Equal(t, 3, total)
	assert.Equal(t, "aaaa1111-0000-0000-0000-000000000003", all[0].ID())

	pending := domainGroup.JoinRequestStatusPending
	onlyPending, total, err := repo.FindByGroup(context.Background(), itGroup1, domainGroup.JoinRequestFilters{Status: &pending})
	require.NoError(t, err)
	assert.Equal(t, 2, total)
	assert.Len(t, onlyPending, 2)

	page, _, err := repo.FindByGroup(context.Background(), itGroup1, domainGroup.JoinRequestFilters{Page: 2, Limit: 2})
	require.NoError(t, err)
	assert.Len(t, page, 1)

	none, total, err := repo.FindByGroup(context.Background(), "not-a-uuid", domainGroup.JoinRequestFilters{})
	require.NoError(t, err)
	assert.Equal(t, 0, total)
	assert.Empty(t, none)
}

func TestJoinRequestRepository_Delete_OnlyWhilePending(t *testing.T) {
	pool := pgtest.NewDB(t)
	itSeedGroupWithOwner(t, pool)
	repo := NewJoinRequestRepository(pool)
	require.NoError(t, repo.Save(context.Background(), itJoinRequest(t, itRequest1, itUserB, nil, itNow)))
	require.NoError(t, repo.Save(context.Background(), domainGroup.RestoreJoinRequest(itRequest2, itGroup1, shared.RestoreUserID(itUserC), nil, domainGroup.JoinRequestStatusApproved, itNow)))

	require.NoError(t, repo.Delete(context.Background(), itRequest1))
	itRequireCode(t, repo.Delete(context.Background(), itRequest1), domainGroup.ErrCodeRequestAlreadyProcessed)
	itRequireCode(t, repo.Delete(context.Background(), itRequest2), domainGroup.ErrCodeRequestAlreadyProcessed)
}

func TestDeletionProvider_GetDeletionCounts(t *testing.T) {
	pool := pgtest.NewDB(t)
	itSeedGroupWithOwner(t, pool)
	pgtest.SeedProblem(t, pool, itProblem1, "two-sum", "PUBLISHED")
	pgtest.SeedGroupMember(t, pool, itMemberID+"01", itGroup1, itUserA, "LEAD")
	pgtest.SeedGroupMember(t, pool, itMemberID+"02", itGroup1, itUserB, "MEMBER")
	now := time.Now()
	pgtest.SeedContest(t, pool, pgtest.Contest{ID: itContest1, GroupID: itGroup1, OwnerID: itUserA, Start: now.Add(-time.Hour), End: now.Add(time.Hour)})
	pgtest.SeedContest(t, pool, pgtest.Contest{ID: itContest2, GroupID: itGroup1, OwnerID: itUserA, Start: now.Add(24 * time.Hour), End: now.Add(26 * time.Hour)})
	pgtest.SeedSubmission(t, pool, pgtest.Submission{ID: "in-contest", ProblemID: itProblem1, UserID: itUserB, ContestID: itContest1})
	pgtest.SeedSubmission(t, pool, pgtest.Submission{ID: "practice", ProblemID: itProblem1, UserID: itUserB})

	got, err := NewDeletionProvider(pool).GetDeletionCounts(context.Background(), itGroup1)

	require.NoError(t, err)
	assert.ElementsMatch(t, []string{itContest1, itContest2}, got.ContestIDs)
	assert.Equal(t, 1, got.ActiveContestsCount)
	assert.Equal(t, 1, got.SubmissionsCount)
	assert.Equal(t, 2, got.MembersCount)
	assert.Equal(t, 0, got.MaterialsCount)

	empty, err := NewDeletionProvider(pool).GetDeletionCounts(context.Background(), itMissing)
	require.NoError(t, err)
	assert.Empty(t, empty.ContestIDs)
	assert.NotNil(t, empty.ContestIDs)
}

func TestDeletionProvider_ListSoleLeadGroups_OnlyWhenNoOtherActiveLeadRemains(t *testing.T) {
	pool := pgtest.NewDB(t)
	itSeedGroupWithOwner(t, pool)
	itSave(t, pool, itGroup(t, itGroup2, "Data Structures", domainGroup.VisibilityVisible, domainGroup.JoinPolicyOpen, itNow))
	itSave(t, pool, itGroup(t, itGroup3, "Graphs", domainGroup.VisibilityVisible, domainGroup.JoinPolicyOpen, itNow))
	pgtest.SeedGroupMember(t, pool, itMemberID+"01", itGroup1, itUserA, "LEAD")
	pgtest.SeedGroupMember(t, pool, itMemberID+"02", itGroup2, itUserA, "LEAD")
	pgtest.SeedGroupMember(t, pool, itMemberID+"03", itGroup2, itUserB, "LEAD")
	pgtest.SeedGroupMember(t, pool, itMemberID+"04", itGroup3, itUserA, "LEAD")
	pgtest.SeedGroupMember(t, pool, itMemberID+"05", itGroup3, itUserC, "LEAD")
	_, err := pool.Exec(context.Background(), `UPDATE users SET status = 'DEACTIVATED', email = NULL, deactivated_at = NOW() WHERE id = $1`, itUserC)
	require.NoError(t, err)

	got, err := NewDeletionProvider(pool).ListSoleLeadGroups(context.Background(), itUserA)

	require.NoError(t, err)
	names := make([]string, len(got))
	for i, g := range got {
		names[i] = g.Name
	}
	assert.Equal(t, []string{"Algorithms", "Graphs"}, names, "group 2 has another active lead, group 3's other lead is deactivated")

	none, err := NewDeletionProvider(pool).ListSoleLeadGroups(context.Background(), itUserB)
	require.NoError(t, err)
	assert.Empty(t, none)
}

func TestContestRegistrationCleaner_DeletesOnlyScheduledRegistrationsOfThatGroupAndUser(t *testing.T) {
	pool := pgtest.NewDB(t)
	itSeedGroupWithOwner(t, pool)
	itSave(t, pool, itGroup(t, itGroup2, "Data Structures", domainGroup.VisibilityVisible, domainGroup.JoinPolicyOpen, itNow))
	now := time.Now()
	pgtest.SeedContest(t, pool, pgtest.Contest{ID: itContest1, GroupID: itGroup1, OwnerID: itUserA, Start: now.Add(24 * time.Hour), End: now.Add(26 * time.Hour)})
	pgtest.SeedContest(t, pool, pgtest.Contest{ID: itContest2, GroupID: itGroup1, OwnerID: itUserA, Start: now.Add(-time.Hour), End: now.Add(time.Hour)})
	pgtest.SeedContest(t, pool, pgtest.Contest{ID: "cccccccc-0000-0000-0000-000000000003", GroupID: itGroup2, OwnerID: itUserA, Start: now.Add(24 * time.Hour), End: now.Add(26 * time.Hour)})
	pgtest.SeedRegistration(t, pool, "ffffffff-0000-0000-0000-000000000001", itContest1, itUserB)
	pgtest.SeedRegistration(t, pool, "ffffffff-0000-0000-0000-000000000002", itContest2, itUserB)
	pgtest.SeedRegistration(t, pool, "ffffffff-0000-0000-0000-000000000003", "cccccccc-0000-0000-0000-000000000003", itUserB)
	pgtest.SeedRegistration(t, pool, "ffffffff-0000-0000-0000-000000000004", itContest1, itUserC)

	deleted, err := NewContestRegistrationCleaner(pool).DeleteScheduledByGroupAndUser(context.Background(), itGroup1, itUserB)

	require.NoError(t, err)
	assert.Equal(t, 1, deleted)
	var remaining int
	require.NoError(t, pool.QueryRow(context.Background(), `SELECT COUNT(*) FROM contest_registrations`).Scan(&remaining))
	assert.Equal(t, 3, remaining)
}

func TestTeamSelectionCleaner_ReducesTheSelectionOrCancelsItBelowTheTeamMinimum(t *testing.T) {
	pool := pgtest.NewDB(t)
	itSeedGroupWithOwner(t, pool)
	now := time.Now()
	pgtest.SeedContest(t, pool, pgtest.Contest{ID: itContest1, GroupID: itGroup1, OwnerID: itUserA, Start: now.Add(24 * time.Hour), End: now.Add(26 * time.Hour)})
	pgtest.SeedContest(t, pool, pgtest.Contest{ID: itContest2, GroupID: itGroup1, OwnerID: itUserA, Start: now.Add(24 * time.Hour), End: now.Add(26 * time.Hour)})
	pgtest.SeedTeam(t, pool, itTeam1, "Team One", itUserA)
	pgtest.SeedTeam(t, pool, "eeeeeeee-0000-0000-0000-000000000002", "Team Two", itUserA)
	pgtest.SeedTeamParticipation(t, pool, "99990000-0000-0000-0000-000000000001", itContest1, itTeam1, itUserA, itUserB, itUserC)
	pgtest.SeedTeamParticipation(t, pool, "99990000-0000-0000-0000-000000000002", itContest2, "eeeeeeee-0000-0000-0000-000000000002", itUserB, itUserC)

	require.NoError(t, NewTeamSelectionCleaner(pool).RemoveFromScheduledByGroupAndUser(context.Background(), itGroup1, itUserC))

	var selected []string
	require.NoError(t, pool.QueryRow(context.Background(),
		`SELECT selected_members FROM contest_team_participants WHERE id = $1`, "99990000-0000-0000-0000-000000000001").Scan(&selected))
	assert.ElementsMatch(t, []string{itUserA, itUserB}, selected, "still meets the minimum of two")
	var cancelled int
	require.NoError(t, pool.QueryRow(context.Background(),
		`SELECT COUNT(*) FROM contest_team_participants WHERE id = $1`, "99990000-0000-0000-0000-000000000002").Scan(&cancelled))
	assert.Equal(t, 0, cancelled, "one member left, below the minimum of two, so the registration is cancelled")
}
