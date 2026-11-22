//go:build integration

package group

import (
	"context"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	domainGroup "github.com/training-judge-center/backend/internal/domain/group"
	"github.com/training-judge-center/backend/internal/domain/shared"
	"github.com/training-judge-center/backend/internal/testutil/pgtest"
)

func itSeedGroupWithOwner(t *testing.T, pool *pgxpool.Pool) {
	t.Helper()
	itSeedUsers(t, pool)
	itSave(t, pool, itGroup(t, itGroup1, "Algorithms", domainGroup.VisibilityVisible, domainGroup.JoinPolicyInvite, itNow))
}

func itMember(t *testing.T, id, groupID, userID string, role domainGroup.MemberRole, addedBy *string, at time.Time) *domainGroup.GroupMember {
	t.Helper()
	var by *shared.UserID
	if addedBy != nil {
		v := shared.RestoreUserID(*addedBy)
		by = &v
	}
	method := domainGroup.JoinMethodOpenJoin
	if by != nil {
		method = domainGroup.JoinMethodDirectAdd
	}
	m, err := domainGroup.NewGroupMember(id, groupID, shared.RestoreUserID(userID), role, method, by, at)
	require.NoError(t, err)
	return m
}

func TestMemberRepository_SaveAndFindByGroupAndUser_RoundTrips(t *testing.T) {
	pool := pgtest.NewDB(t)
	itSeedGroupWithOwner(t, pool)
	repo := NewMemberRepository(pool)
	adder := itUserA
	require.NoError(t, repo.Save(context.Background(), itMember(t, itMemberID+"01", itGroup1, itUserB, domainGroup.MemberRoleLead, &adder, itNow)))

	got, err := repo.FindByGroupAndUser(context.Background(), itGroup1, shared.RestoreUserID(itUserB))

	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, "LEAD", got.Role().String())
	assert.Equal(t, "DIRECT_ADD", got.JoinMethod().String())
	require.NotNil(t, got.AddedBy())
	assert.Equal(t, itUserA, got.AddedBy().Value())
	assert.True(t, itNow.Equal(got.JoinedAt()))

	missing, err := repo.FindByGroupAndUser(context.Background(), itGroup1, shared.RestoreUserID(itUserC))
	require.NoError(t, err)
	assert.Nil(t, missing)
}

func TestMemberRepository_Save_SecondMembershipReturnsAlreadyMember(t *testing.T) {
	pool := pgtest.NewDB(t)
	itSeedGroupWithOwner(t, pool)
	repo := NewMemberRepository(pool)
	require.NoError(t, repo.Save(context.Background(), itMember(t, itMemberID+"01", itGroup1, itUserB, domainGroup.MemberRoleMember, nil, itNow)))

	err := repo.Save(context.Background(), itMember(t, itMemberID+"02", itGroup1, itUserB, domainGroup.MemberRoleLead, nil, itNow))

	itRequireCode(t, err, domainGroup.ErrCodeAlreadyMember)
}

func TestMemberRepository_UpdateAndDelete_ChangeTheRoleOrRemoveAndReportNotAMember(t *testing.T) {
	pool := pgtest.NewDB(t)
	itSeedGroupWithOwner(t, pool)
	repo := NewMemberRepository(pool)
	member := itMember(t, itMemberID+"01", itGroup1, itUserB, domainGroup.MemberRoleMember, nil, itNow)
	require.NoError(t, repo.Save(context.Background(), member))

	promoted := itMember(t, itMemberID+"01", itGroup1, itUserB, domainGroup.MemberRoleLead, nil, itNow)
	require.NoError(t, repo.Update(context.Background(), promoted))
	got, err := repo.FindByGroupAndUser(context.Background(), itGroup1, shared.RestoreUserID(itUserB))
	require.NoError(t, err)
	assert.Equal(t, "LEAD", got.Role().String())

	stranger := itMember(t, itMemberID+"09", itGroup1, itUserC, domainGroup.MemberRoleLead, nil, itNow)
	itRequireCode(t, repo.Update(context.Background(), stranger), domainGroup.ErrCodeNotAMember)

	require.NoError(t, repo.Delete(context.Background(), itGroup1, shared.RestoreUserID(itUserB)))
	itRequireCode(t, repo.Delete(context.Background(), itGroup1, shared.RestoreUserID(itUserB)), domainGroup.ErrCodeNotAMember)
}

func TestMemberRepository_SaveAll_StopsAtTheFirstFailure(t *testing.T) {
	pool := pgtest.NewDB(t)
	itSeedGroupWithOwner(t, pool)
	repo := NewMemberRepository(pool)
	members := []*domainGroup.GroupMember{
		itMember(t, itMemberID+"01", itGroup1, itUserB, domainGroup.MemberRoleMember, nil, itNow),
		itMember(t, itMemberID+"02", itGroup1, itUserB, domainGroup.MemberRoleMember, nil, itNow),
		itMember(t, itMemberID+"03", itGroup1, itUserC, domainGroup.MemberRoleMember, nil, itNow),
	}

	err := repo.SaveAll(context.Background(), members)

	itRequireCode(t, err, domainGroup.ErrCodeAlreadyMember)
	count, err := repo.CountMembers(context.Background(), itGroup1)
	require.NoError(t, err)
	assert.Equal(t, 1, count, "SaveAll alone is not transactional; the caller wraps it")
}

func TestMemberRepository_FindByGroup_FiltersByRoleAndPaginatesOldestFirst(t *testing.T) {
	pool := pgtest.NewDB(t)
	itSeedGroupWithOwner(t, pool)
	repo := NewMemberRepository(pool)
	require.NoError(t, repo.Save(context.Background(), itMember(t, itMemberID+"02", itGroup1, itUserB, domainGroup.MemberRoleMember, nil, itNow.Add(time.Hour))))
	require.NoError(t, repo.Save(context.Background(), itMember(t, itMemberID+"01", itGroup1, itUserA, domainGroup.MemberRoleLead, nil, itNow)))
	require.NoError(t, repo.Save(context.Background(), itMember(t, itMemberID+"03", itGroup1, itUserC, domainGroup.MemberRoleMember, nil, itNow.Add(2*time.Hour))))

	all, total, err := repo.FindByGroup(context.Background(), itGroup1, domainGroup.MemberFilters{Page: 1, Limit: 2})
	require.NoError(t, err)
	assert.Equal(t, 3, total)
	require.Len(t, all, 2)
	assert.Equal(t, itUserA, all[0].UserID().Value())
	assert.Equal(t, itUserB, all[1].UserID().Value())

	lead := domainGroup.MemberRoleLead
	leads, total, err := repo.FindByGroup(context.Background(), itGroup1, domainGroup.MemberFilters{Role: &lead, Page: 1})
	require.NoError(t, err)
	assert.Equal(t, 1, total)
	require.Len(t, leads, 1)
	assert.Equal(t, itUserA, leads[0].UserID().Value())

	count, err := repo.CountMembers(context.Background(), itGroup1)
	require.NoError(t, err)
	assert.Equal(t, 3, count)
	leadCount, err := repo.CountLeads(context.Background(), itGroup1)
	require.NoError(t, err)
	assert.Equal(t, 1, leadCount)

	listed, err := repo.ListLeads(context.Background(), itGroup1)
	require.NoError(t, err)
	require.Len(t, listed, 1)
	assert.Equal(t, itUserA, listed[0].UserID().Value())
}

func TestMemberRepository_BulkStats_CountsAndTheViewersMembership(t *testing.T) {
	pool := pgtest.NewDB(t)
	itSeedGroupWithOwner(t, pool)
	itSave(t, pool, itGroup(t, itGroup2, "Data Structures", domainGroup.VisibilityVisible, domainGroup.JoinPolicyOpen, itNow))
	repo := NewMemberRepository(pool)
	require.NoError(t, repo.Save(context.Background(), itMember(t, itMemberID+"01", itGroup1, itUserA, domainGroup.MemberRoleLead, nil, itNow)))
	require.NoError(t, repo.Save(context.Background(), itMember(t, itMemberID+"02", itGroup1, itUserB, domainGroup.MemberRoleMember, nil, itNow)))

	stats, err := repo.BulkStats(context.Background(), []string{itGroup1, itGroup2}, shared.RestoreUserID(itUserB))

	require.NoError(t, err)
	assert.Equal(t, 2, stats[itGroup1].Count)
	assert.True(t, stats[itGroup1].IsMember)
	assert.Equal(t, "MEMBER", stats[itGroup1].Role.String())
	assert.Equal(t, 0, stats[itGroup2].Count)
	assert.False(t, stats[itGroup2].IsMember)

	empty, err := repo.BulkStats(context.Background(), nil, shared.RestoreUserID(itUserB))
	require.NoError(t, err)
	assert.Empty(t, empty)
}

func itInvitation(t *testing.T, id string, invitee *string, at time.Time) *domainGroup.GroupInvitation {
	t.Helper()
	var inv *shared.UserID
	if invitee != nil {
		v := shared.RestoreUserID(*invitee)
		inv = &v
	}
	g, err := domainGroup.NewGroupInvitation(id, itGroup1, inv, shared.RestoreUserID(itUserA), at)
	require.NoError(t, err)
	return g
}

func TestInvitationRepository_SaveAndFindByID_RoundTrips(t *testing.T) {
	pool := pgtest.NewDB(t)
	itSeedGroupWithOwner(t, pool)
	repo := NewInvitationRepository(pool)
	invitee := itUserB
	require.NoError(t, repo.Save(context.Background(), itInvitation(t, "aaaa0000-0000-0000-0000-000000000001", &invitee, itNow)))

	got, err := repo.FindByID(context.Background(), "aaaa0000-0000-0000-0000-000000000001")

	require.NoError(t, err)
	require.NotNil(t, got.InviteeID())
	assert.Equal(t, itUserB, got.InviteeID().Value())
	assert.Equal(t, itUserA, got.InvitedBy().Value())
	assert.Equal(t, "PENDING", got.Status().String())
	assert.True(t, got.ExpiresAt().After(got.CreatedAt()))

	_, err = repo.FindByID(context.Background(), itMissing)
	itRequireCode(t, err, domainGroup.ErrCodeInvitationNotFound)
	_, err = repo.FindByID(context.Background(), "not-a-uuid")
	itRequireCode(t, err, domainGroup.ErrCodeInvitationNotFound)
}

func TestInvitationRepository_Save_SecondPendingForTheSameInviteeIsRefused(t *testing.T) {
	pool := pgtest.NewDB(t)
	itSeedGroupWithOwner(t, pool)
	repo := NewInvitationRepository(pool)
	invitee := itUserB
	require.NoError(t, repo.Save(context.Background(), itInvitation(t, "aaaa0000-0000-0000-0000-000000000001", &invitee, itNow)))

	err := repo.Save(context.Background(), itInvitation(t, "aaaa0000-0000-0000-0000-000000000002", &invitee, itNow))

	itRequireCode(t, err, domainGroup.ErrCodeInvitationAlreadyProcessed)
}

func TestInvitationRepository_Save_OnlyOneGeneralPendingInvitationPerGroup(t *testing.T) {
	pool := pgtest.NewDB(t)
	itSeedGroupWithOwner(t, pool)
	repo := NewInvitationRepository(pool)
	require.NoError(t, repo.Save(context.Background(), itInvitation(t, "aaaa0000-0000-0000-0000-000000000001", nil, itNow)))

	err := repo.Save(context.Background(), itInvitation(t, "aaaa0000-0000-0000-0000-000000000002", nil, itNow))

	itRequireCode(t, err, domainGroup.ErrCodeInvitationAlreadyProcessed)
}

func TestInvitationRepository_FindPendingByGroupAndInvitee_PerInviteeAndGeneral(t *testing.T) {
	pool := pgtest.NewDB(t)
	itSeedGroupWithOwner(t, pool)
	repo := NewInvitationRepository(pool)
	invitee := itUserB
	require.NoError(t, repo.Save(context.Background(), itInvitation(t, "aaaa0000-0000-0000-0000-000000000001", &invitee, itNow)))
	require.NoError(t, repo.Save(context.Background(), itInvitation(t, "aaaa0000-0000-0000-0000-000000000002", nil, itNow)))
	userB := shared.RestoreUserID(itUserB)
	userC := shared.RestoreUserID(itUserC)

	personal, err := repo.FindPendingByGroupAndInvitee(context.Background(), itGroup1, &userB)
	require.NoError(t, err)
	require.NotNil(t, personal)
	assert.Equal(t, "aaaa0000-0000-0000-0000-000000000001", personal.ID())

	general, err := repo.FindPendingByGroupAndInvitee(context.Background(), itGroup1, nil)
	require.NoError(t, err)
	require.NotNil(t, general)
	assert.Equal(t, "aaaa0000-0000-0000-0000-000000000002", general.ID())

	none, err := repo.FindPendingByGroupAndInvitee(context.Background(), itGroup1, &userC)
	require.NoError(t, err)
	assert.Nil(t, none)
}

func TestInvitationRepository_TransitionStatus_OnlyFromTheExpectedState(t *testing.T) {
	pool := pgtest.NewDB(t)
	itSeedGroupWithOwner(t, pool)
	repo := NewInvitationRepository(pool)
	invitee := itUserB
	id := "aaaa0000-0000-0000-0000-000000000001"
	require.NoError(t, repo.Save(context.Background(), itInvitation(t, id, &invitee, itNow)))

	require.NoError(t, repo.TransitionStatus(context.Background(), id, domainGroup.InvitationStatusPending, domainGroup.InvitationStatusAccepted))
	got, err := repo.FindByID(context.Background(), id)
	require.NoError(t, err)
	assert.Equal(t, "ACCEPTED", got.Status().String())

	err = repo.TransitionStatus(context.Background(), id, domainGroup.InvitationStatusPending, domainGroup.InvitationStatusRevoked)
	itRequireCode(t, err, domainGroup.ErrCodeInvitationAlreadyProcessed)
	again, err := repo.FindByID(context.Background(), id)
	require.NoError(t, err)
	assert.Equal(t, "ACCEPTED", again.Status().String())
}

func TestInvitationRepository_FindByGroup_FiltersByStatusAndPaginatesNewestFirst(t *testing.T) {
	pool := pgtest.NewDB(t)
	itSeedGroupWithOwner(t, pool)
	repo := NewInvitationRepository(pool)
	b, c := itUserB, itUserC
	require.NoError(t, repo.Save(context.Background(), itInvitation(t, "aaaa0000-0000-0000-0000-000000000001", &b, itNow)))
	require.NoError(t, repo.Save(context.Background(), itInvitation(t, "aaaa0000-0000-0000-0000-000000000002", &c, itNow.Add(time.Hour))))
	require.NoError(t, repo.Save(context.Background(), itInvitation(t, "aaaa0000-0000-0000-0000-000000000003", nil, itNow.Add(2*time.Hour))))
	require.NoError(t, repo.TransitionStatus(context.Background(), "aaaa0000-0000-0000-0000-000000000001", domainGroup.InvitationStatusPending, domainGroup.InvitationStatusRevoked))

	all, total, err := repo.FindByGroup(context.Background(), itGroup1, domainGroup.InvitationFilters{})
	require.NoError(t, err)
	assert.Equal(t, 3, total)
	require.Len(t, all, 3)
	assert.Equal(t, "aaaa0000-0000-0000-0000-000000000003", all[0].ID())

	revoked := domainGroup.InvitationStatusRevoked
	only, total, err := repo.FindByGroup(context.Background(), itGroup1, domainGroup.InvitationFilters{Status: &revoked})
	require.NoError(t, err)
	assert.Equal(t, 1, total)
	require.Len(t, only, 1)

	page, total, err := repo.FindByGroup(context.Background(), itGroup1, domainGroup.InvitationFilters{Page: 2, Limit: 2})
	require.NoError(t, err)
	assert.Equal(t, 3, total)
	require.Len(t, page, 1)

	none, total, err := repo.FindByGroup(context.Background(), "not-a-uuid", domainGroup.InvitationFilters{})
	require.NoError(t, err)
	assert.Equal(t, 0, total)
	assert.Empty(t, none)
}
