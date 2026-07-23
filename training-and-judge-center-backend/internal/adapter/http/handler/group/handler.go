package group

import (
	appGroup "github.com/training-judge-center/backend/internal/application/group"
)

type Handler struct {
	createGroup          *appGroup.CreateGroupUseCase
	listGroups           *appGroup.ListGroupsUseCase
	getGroup             *appGroup.GetGroupUseCase
	listMyGroups         *appGroup.ListMyGroupsUseCase
	joinGroup            *appGroup.JoinGroupUseCase
	requestJoin          *appGroup.RequestJoinUseCase
	approveRequest       *appGroup.ApproveRequestUseCase
	rejectRequest        *appGroup.RejectRequestUseCase
	listRequests         *appGroup.ListJoinRequestsUseCase
	getMyRequest         *appGroup.GetMyRequestUseCase
	cancelMyRequest      *appGroup.CancelMyRequestUseCase
	generateInvite       *appGroup.GenerateInviteUseCase
	acceptInvite         *appGroup.AcceptInviteUseCase
	listGroupInvitations *appGroup.ListGroupInvitationsUseCase
	revokeInvitation     *appGroup.RevokeInvitationUseCase
	inviteByNicknames    *appGroup.InviteByNicknamesUseCase
	addMember            *appGroup.AddMemberUseCase
	removeMember         *appGroup.RemoveMemberUseCase
	changeRole           *appGroup.ChangeRoleUseCase
	leaveGroup           *appGroup.LeaveGroupUseCase
	listMembers          *appGroup.ListMembersUseCase
	deleteGroup          *appGroup.DeleteGroupUseCase
	updateGroup          *appGroup.UpdateGroupUseCase
}

func NewHandler(
	createGroup *appGroup.CreateGroupUseCase,
	listGroups *appGroup.ListGroupsUseCase,
	getGroup *appGroup.GetGroupUseCase,
	listMyGroups *appGroup.ListMyGroupsUseCase,
	joinGroup *appGroup.JoinGroupUseCase,
	requestJoin *appGroup.RequestJoinUseCase,
	approveRequest *appGroup.ApproveRequestUseCase,
	rejectRequest *appGroup.RejectRequestUseCase,
	listRequests *appGroup.ListJoinRequestsUseCase,
	getMyRequest *appGroup.GetMyRequestUseCase,
	cancelMyRequest *appGroup.CancelMyRequestUseCase,
	generateInvite *appGroup.GenerateInviteUseCase,
	acceptInvite *appGroup.AcceptInviteUseCase,
	listGroupInvitations *appGroup.ListGroupInvitationsUseCase,
	revokeInvitation *appGroup.RevokeInvitationUseCase,
	inviteByNicknames *appGroup.InviteByNicknamesUseCase,
	addMember *appGroup.AddMemberUseCase,
	removeMember *appGroup.RemoveMemberUseCase,
	changeRole *appGroup.ChangeRoleUseCase,
	leaveGroup *appGroup.LeaveGroupUseCase,
	listMembers *appGroup.ListMembersUseCase,
	deleteGroup *appGroup.DeleteGroupUseCase,
	updateGroup *appGroup.UpdateGroupUseCase,
) *Handler {
	return &Handler{
		createGroup:          createGroup,
		listGroups:           listGroups,
		getGroup:             getGroup,
		listMyGroups:         listMyGroups,
		joinGroup:            joinGroup,
		requestJoin:          requestJoin,
		approveRequest:       approveRequest,
		rejectRequest:        rejectRequest,
		listRequests:         listRequests,
		getMyRequest:         getMyRequest,
		cancelMyRequest:      cancelMyRequest,
		generateInvite:       generateInvite,
		acceptInvite:         acceptInvite,
		listGroupInvitations: listGroupInvitations,
		revokeInvitation:     revokeInvitation,
		inviteByNicknames:    inviteByNicknames,
		addMember:            addMember,
		removeMember:         removeMember,
		changeRole:           changeRole,
		leaveGroup:           leaveGroup,
		listMembers:          listMembers,
		deleteGroup:          deleteGroup,
		updateGroup:          updateGroup,
	}
}
