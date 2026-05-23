package group

import (
	"context"
	"log/slog"

	appshared "github.com/training-judge-center/backend/internal/application/shared"
	domainGroup "github.com/training-judge-center/backend/internal/domain/group"
	"github.com/training-judge-center/backend/pkg/apperror"
)

const maxRequestsPageLimit = 100

type ListJoinRequestsInput struct {
	GroupID     string
	Status      string
	Page        int
	Limit       int
	CurrentUser appshared.CurrentUser
}

type JoinRequestDetail struct {
	Request JoinRequestDTO
	Display *UserDisplay
}

type ListJoinRequestsOutput struct {
	Requests   []JoinRequestDetail
	Total      int
	TotalPages int
}

type ListJoinRequestsUseCase struct {
	memberRepo      domainGroup.MemberRepository
	joinRequestRepo domainGroup.JoinRequestRepository
	userProvider    UserProvider
}

func NewListJoinRequestsUseCase(
	memberRepo domainGroup.MemberRepository,
	joinRequestRepo domainGroup.JoinRequestRepository,
	userProvider UserProvider,
) *ListJoinRequestsUseCase {
	return &ListJoinRequestsUseCase{memberRepo: memberRepo, joinRequestRepo: joinRequestRepo, userProvider: userProvider}
}

func (uc *ListJoinRequestsUseCase) Execute(ctx context.Context, input ListJoinRequestsInput) (*ListJoinRequestsOutput, error) {
	if err := requireLeadOrAdmin(ctx, uc.memberRepo, input.GroupID, input.CurrentUser); err != nil {
		return nil, err
	}

	if err := appshared.ValidatePagination(input.Page, input.Limit, maxRequestsPageLimit); err != nil {
		return nil, err
	}
	page := input.Page
	limit := input.Limit

	var statusFilter *domainGroup.JoinRequestStatus
	if input.Status != "" {
		s, err := domainGroup.NewJoinRequestStatus(input.Status)
		if err != nil {
			return nil, apperror.NewValidation([]apperror.FieldError{
				{Field: "status", Message: "invalid status filter; must be PENDING, APPROVED, or REJECTED"},
			})
		}
		statusFilter = &s
	} else {
		pending := domainGroup.JoinRequestStatusPending
		statusFilter = &pending
	}

	requests, total, err := uc.joinRequestRepo.FindByGroup(ctx, input.GroupID, domainGroup.JoinRequestFilters{
		Status: statusFilter,
		Page:   page,
		Limit:  limit,
	})
	if err != nil {
		return nil, err
	}

	userIDs := make([]string, 0, len(requests))
	for _, r := range requests {
		userIDs = append(userIDs, r.RequesterUserID().Value())
	}
	displays, err := uc.userProvider.GetDisplays(ctx, userIDs)
	if err != nil {
		slog.ErrorContext(ctx, "failed to get user displays for join requests", "error", err)
		return nil, apperror.NewInternal()
	}

	details := make([]JoinRequestDetail, 0, len(requests))
	for _, r := range requests {
		d := displays[r.RequesterUserID().Value()]
		if d == nil {
			slog.ErrorContext(ctx, "user display not found for join request", "user_id", r.RequesterUserID().Value())
			d = &UserDisplay{Nickname: "unknown"}
		}
		details = append(details, JoinRequestDetail{Request: joinRequestToDTO(r), Display: d})
	}

	return &ListJoinRequestsOutput{
		Requests:   details,
		Total:      total,
		TotalPages: appshared.CalcTotalPages(total, limit),
	}, nil
}
