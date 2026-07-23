package group

type paginationResp struct {
	TotalCount   int  `json:"total"`
	CurrentPage  int  `json:"page"`
	TotalPages   int  `json:"totalPages"`
	ItemsPerPage int  `json:"limit"`
	HasNextPage  bool `json:"hasNextPage"`
	HasPrevPage  bool `json:"hasPrevPage"`
}

type groupListItemResp struct {
	ID          string  `json:"id"`
	Name        string  `json:"name"`
	Description *string `json:"description"`
	Visibility  string  `json:"visibility"`
	JoinPolicy  string  `json:"joinPolicy"`
	IsGlobal    bool    `json:"isGlobal"`
	MemberCount int     `json:"memberCount"`
	UserRole    *string `json:"userRole"`
	CreatedAt   string  `json:"createdAt"`
}

type listGroupsResponse struct {
	Groups     []groupListItemResp `json:"groups"`
	Pagination paginationResp      `json:"pagination"`
}

type leadResp struct {
	UserID   string `json:"userId"`
	Nickname string `json:"nickname"`
	Name     string `json:"name"`
}

type statisticsResp struct {
	MemberCount int `json:"memberCount"`
	LeadCount   int `json:"leadCount"`
}

type userMembershipResp struct {
	IsMember bool    `json:"isMember"`
	Role     *string `json:"role"`
	JoinedAt *string `json:"joinedAt"`
}

type getGroupResponse struct {
	ID             string             `json:"id"`
	Name           string             `json:"name"`
	Description    *string            `json:"description"`
	Visibility     string             `json:"visibility"`
	JoinPolicy     string             `json:"joinPolicy"`
	IsGlobal       bool               `json:"isGlobal"`
	Statistics     statisticsResp     `json:"statistics"`
	Leads          []leadResp         `json:"leads"`
	UserMembership userMembershipResp `json:"userMembership"`
	CreatedAt      string             `json:"createdAt"`
	UpdatedAt      string             `json:"updatedAt"`
}

type myGroupItemResp struct {
	ID          string  `json:"id"`
	Name        string  `json:"name"`
	Description *string `json:"description"`
	Visibility  string  `json:"visibility"`
	JoinPolicy  string  `json:"joinPolicy"`
	IsGlobal    bool    `json:"isGlobal"`
	MyRole      string  `json:"myRole"`
	JoinedAt    string  `json:"joinedAt"`
	MemberCount int     `json:"memberCount"`
	CreatedAt   string  `json:"createdAt"`
}

type listMyGroupsResponse struct {
	Groups     []myGroupItemResp `json:"groups"`
	Pagination paginationResp    `json:"pagination"`
}

type requesterResp struct {
	UserID   string `json:"userId"`
	Nickname string `json:"nickname"`
	Name     string `json:"name"`
	Email    string `json:"email"`
}

type joinRequestResp struct {
	ID        string         `json:"id"`
	GroupID   string         `json:"groupId"`
	Requester *requesterResp `json:"requester,omitempty"`
	Status    string         `json:"status"`
	Message   *string        `json:"message,omitempty"`
	CreatedAt string         `json:"createdAt"`
}

type listRequestsResponse struct {
	Requests   []joinRequestResp `json:"requests"`
	Pagination paginationResp    `json:"pagination"`
}

func buildPagination(total, page, totalPages, limit int) paginationResp {
	return paginationResp{
		TotalCount:   total,
		CurrentPage:  page,
		TotalPages:   totalPages,
		ItemsPerPage: limit,
		HasNextPage:  page < totalPages,
		HasPrevPage:  page > 1 && total > 0,
	}
}

// ── Member endpoints ──────────────────────────────────────────────────────────

type addMemberReq struct {
	Nickname string `json:"nickname"`
	Role     string `json:"role"`
}

type addMemberResp struct {
	GroupID    string `json:"groupId"`
	UserID     string `json:"userId"`
	Nickname   string `json:"nickname"`
	Role       string `json:"role"`
	JoinedAt   string `json:"joinedAt"`
	AddedBy    string `json:"addedBy"`
	JoinMethod string `json:"joinMethod"`
}

type changeRoleReq struct {
	Role string `json:"role"`
}

type changeRoleResp struct {
	GroupID       string `json:"groupId"`
	UserID        string `json:"userId"`
	Nickname      string `json:"nickname"`
	Role          string `json:"role"`
	JoinedAt      string `json:"joinedAt"`
	RoleChangedAt string `json:"roleChangedAt"`
}

type memberListItemResp struct {
	UserID   string `json:"userId"`
	Nickname string `json:"nickname"`
	Name     string `json:"name"`
	Role     string `json:"role"`
	JoinedAt string `json:"joinedAt"`
}

type listMembersResp struct {
	Members    []memberListItemResp `json:"members"`
	Pagination paginationResp       `json:"pagination"`
}

// ── Update group endpoint ─────────────────────────────────────────────────────

type updateGroupReq struct {
	Name        *string `json:"name"`
	Description *string `json:"description"`
	Visibility  *string `json:"visibility"`
	JoinPolicy  *string `json:"joinPolicy"`
}

type policyChangeEffectsResp struct {
	RequestsAutoApproved int `json:"requestsAutoApproved"`
	RequestsAutoRejected int `json:"requestsAutoRejected"`
}

type updateGroupResp struct {
	ID                  string                   `json:"id"`
	Name                string                   `json:"name"`
	Description         *string                  `json:"description"`
	Visibility          string                   `json:"visibility"`
	JoinPolicy          string                   `json:"joinPolicy"`
	CreatedBy           string                   `json:"createdBy"`
	CreatedAt           string                   `json:"createdAt"`
	UpdatedAt           string                   `json:"updatedAt"`
	MembersCount        int                      `json:"membersCount"`
	PolicyChangeEffects *policyChangeEffectsResp `json:"policyChangeEffects,omitempty"`
}

// ── Invitation endpoints ──────────────────────────────────────────────────────

type generateInviteReq struct {
	UserNickname string `json:"userNickname"`
	UserEmail    string `json:"userEmail"`
	UserID       string `json:"userId"`
}

type generateInviteResponse struct {
	ID        string         `json:"id"`
	GroupID   string         `json:"groupId"`
	Invitee   *requesterResp `json:"invitee,omitempty"`
	Status    string         `json:"status"`
	ExpiresAt string         `json:"expiresAt"`
	CreatedAt string         `json:"createdAt"`
}

type acceptInviteRequest struct {
	InvitationID string `json:"invitationId"`
}

type invitationListItemResp struct {
	ID              string         `json:"id"`
	GroupID         string         `json:"groupId"`
	Invitee         *requesterResp `json:"invitee,omitempty"`
	InvitedBy       string         `json:"invitedBy"`
	Status          string         `json:"status"`
	EffectiveStatus string         `json:"effectiveStatus"`
	ExpiresAt       string         `json:"expiresAt"`
	CreatedAt       string         `json:"createdAt"`
}

type listGroupInvitationsResponse struct {
	Invitations []invitationListItemResp `json:"invitations"`
	Pagination  paginationResp           `json:"pagination"`
}

type inviteByNicknamesReq struct {
	Nicknames []string `json:"nicknames"`
}

type invitationSummaryResp struct {
	ID        string `json:"id"`
	Status    string `json:"status"`
	ExpiresAt string `json:"expiresAt"`
	CreatedAt string `json:"createdAt"`
}

type inviteByNicknamesResultResp struct {
	Nickname   string                 `json:"nickname"`
	Status     string                 `json:"status"`
	Invitation *invitationSummaryResp `json:"invitation,omitempty"`
	Invitee    *requesterResp         `json:"invitee,omitempty"`
	Reason     string                 `json:"reason,omitempty"`
}

type inviteByNicknamesResponse struct {
	Results []inviteByNicknamesResultResp `json:"results"`
}

// ── Delete group endpoint ─────────────────────────────────────────────────────

type deleteGroupReq struct {
	ConfirmationName string `json:"confirmationName"`
}

type deletedGroupInfo struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type deletionSummary struct {
	ContestsDeleted            int `json:"contestsDeleted"`
	MaterialsDeleted           int `json:"materialsDeleted"`
	StandingCollectionsDeleted int `json:"standingCollectionsDeleted"`
	SubmissionsOrphaned        int `json:"submissionsOrphaned"`
	MembersRemoved             int `json:"membersRemoved"`
}

type deleteGroupResp struct {
	Message         string           `json:"message"`
	DeletedGroup    deletedGroupInfo `json:"deletedGroup"`
	DeletionSummary deletionSummary  `json:"deletionSummary"`
}
