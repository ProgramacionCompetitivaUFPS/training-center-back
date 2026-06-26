package contest

import (
	"context"
	"time"

	appshared "github.com/training-judge-center/backend/internal/application/shared"
	domainContest "github.com/training-judge-center/backend/internal/domain/contest"
	"github.com/training-judge-center/backend/pkg/apperror"
)

type ListContestRegistrationsInput struct {
	CurrentUser appshared.CurrentUser
	GroupID     string
	ContestID   string
	Page        int
	Limit       int
}

type RegistrationEntry struct {
	Nickname     string
	RegisteredAt time.Time
}

type ListContestRegistrationsOutput struct {
	Registrations []RegistrationEntry
	Total         int
	Page          int
	Limit         int
}

type ListContestRegistrationsUseCase struct {
	repo             domainContest.Repository
	registrationRepo domainContest.RegistrationRepository
	memberProvider   GroupMemberProvider
	nicknameProvider ParticipantNicknameProvider
}

func NewListContestRegistrationsUseCase(
	repo domainContest.Repository,
	registrationRepo domainContest.RegistrationRepository,
	memberProvider GroupMemberProvider,
	nicknameProvider ParticipantNicknameProvider,
) *ListContestRegistrationsUseCase {
	return &ListContestRegistrationsUseCase{
		repo:             repo,
		registrationRepo: registrationRepo,
		memberProvider:   memberProvider,
		nicknameProvider: nicknameProvider,
	}
}

func (uc *ListContestRegistrationsUseCase) Execute(ctx context.Context, in ListContestRegistrationsInput) (*ListContestRegistrationsOutput, error) {
	c, err := uc.repo.FindByID(ctx, in.ContestID)
	if err != nil {
		return nil, err
	}
	if c == nil {
		return nil, apperror.NewNotFound(domainContest.ErrCodeContestNotFound, "contest not found")
	}

	if c.GroupID().Value() != in.GroupID {
		return nil, apperror.NewNotFound(domainContest.ErrCodeContestNotFound, "contest not found")
	}

	role, err := uc.memberProvider.GetMemberRole(ctx, in.CurrentUser.ID, in.GroupID)
	if err != nil {
		return nil, err
	}
	if role == nil && !in.CurrentUser.IsAdmin() {
		return nil, apperror.NewForbidden(ErrCodeNotGroupMember, "only group members can view contest registrations")
	}

	regs, total, err := uc.registrationRepo.ListByContest(ctx, in.ContestID, in.Page, in.Limit)
	if err != nil {
		return nil, err
	}

	userIDs := make([]string, len(regs))
	for i, r := range regs {
		userIDs[i] = r.UserID()
	}

	nicknames, err := uc.nicknameProvider.GetNicknamesByIDs(ctx, userIDs)
	if err != nil {
		return nil, err
	}

	entries := make([]RegistrationEntry, len(regs))
	for i, r := range regs {
		entries[i] = RegistrationEntry{
			Nickname:     nicknames[r.UserID()],
			RegisteredAt: r.RegisteredAt(),
		}
	}

	return &ListContestRegistrationsOutput{
		Registrations: entries,
		Total:         total,
		Page:          in.Page,
		Limit:         in.Limit,
	}, nil
}
