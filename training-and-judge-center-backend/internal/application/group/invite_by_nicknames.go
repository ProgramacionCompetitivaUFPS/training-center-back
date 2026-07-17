package group

import (
	"context"
	"fmt"
	"strings"
	"time"

	appshared "github.com/training-judge-center/backend/internal/application/shared"
	domainGroup "github.com/training-judge-center/backend/internal/domain/group"
	"github.com/training-judge-center/backend/internal/domain/shared"
	"github.com/training-judge-center/backend/pkg/apperror"
)

// MaxInviteBatchSize caps nicknames per request: each item opens its own
// transaction plus a synchronous SMTP call, so an unbounded batch would make
// a single request slow and costly.
const MaxInviteBatchSize = 50

type InviteResultStatus string

const (
	InviteResultInvited InviteResultStatus = "invited"
	// InviteResultEmailFailed: the invitation WAS persisted, but the email
	// could not be delivered — it still exists and can be accepted via its link.
	InviteResultEmailFailed InviteResultStatus = "email_failed"
	// InviteResultFailed: no invitation exists for this nickname (unknown
	// nickname or a persistence failure).
	InviteResultFailed InviteResultStatus = "failed"
)

type InviteByNicknamesInput struct {
	GroupID     string
	Nicknames   []string
	CurrentUser appshared.CurrentUser
}

type InviteByNicknamesResult struct {
	Nickname   string
	Status     InviteResultStatus
	Invitation *GroupInvitationDTO // nil only when Status == InviteResultFailed
	Invitee    *UserDisplay        // nil only when Status == InviteResultFailed
	Reason     string              // populated only when Status != InviteResultInvited
}

type InviteByNicknamesOutput struct {
	Results []InviteByNicknamesResult
}

type InviteByNicknamesUseCase struct {
	groupRepo        domainGroup.Repository
	memberRepo       domainGroup.MemberRepository
	invitationRepo   domainGroup.InvitationRepository
	nicknameResolver NicknameResolver
	txManager        appshared.TransactionManager
	emailSender      appshared.EmailSender
	frontendBaseURL  string
}

func NewInviteByNicknamesUseCase(
	groupRepo domainGroup.Repository,
	memberRepo domainGroup.MemberRepository,
	invitationRepo domainGroup.InvitationRepository,
	nicknameResolver NicknameResolver,
	txManager appshared.TransactionManager,
	emailSender appshared.EmailSender,
	frontendBaseURL string,
) *InviteByNicknamesUseCase {
	return &InviteByNicknamesUseCase{
		groupRepo:        groupRepo,
		memberRepo:       memberRepo,
		invitationRepo:   invitationRepo,
		nicknameResolver: nicknameResolver,
		txManager:        txManager,
		emailSender:      emailSender,
		frontendBaseURL:  frontendBaseURL,
	}
}

func (uc *InviteByNicknamesUseCase) Execute(ctx context.Context, input InviteByNicknamesInput) (*InviteByNicknamesOutput, error) {
	var fieldErrs []apperror.FieldError
	if input.GroupID == "" {
		fieldErrs = append(fieldErrs, apperror.FieldError{Field: "groupId", Message: "groupId is required"})
	}
	if len(input.Nicknames) == 0 {
		fieldErrs = append(fieldErrs, apperror.FieldError{Field: "nicknames", Message: "at least one nickname is required"})
	}
	if len(input.Nicknames) > MaxInviteBatchSize {
		fieldErrs = append(fieldErrs, apperror.FieldError{Field: "nicknames", Message: fmt.Sprintf("at most %d nicknames are allowed per request", MaxInviteBatchSize)})
	}
	for i, nickname := range input.Nicknames {
		if strings.TrimSpace(nickname) == "" {
			fieldErrs = append(fieldErrs, apperror.FieldError{Field: fmt.Sprintf("nicknames[%d]", i), Message: "nickname must not be blank"})
		}
	}
	if len(fieldErrs) > 0 {
		return nil, apperror.NewValidation(fieldErrs)
	}

	g, err := uc.groupRepo.FindByID(ctx, input.GroupID)
	if err != nil {
		return nil, err
	}

	if err := requireLeadOrAdmin(ctx, uc.memberRepo, input.GroupID, input.CurrentUser); err != nil {
		return nil, err
	}

	if g.JoinPolicy() != domainGroup.JoinPolicyInvite {
		return nil, apperror.NewBadRequest(ErrCodeInvalidJoinPolicy, "this group does not use invite mode")
	}

	invitedBy := shared.RestoreUserID(input.CurrentUser.ID)
	now := time.Now()

	seen := make(map[string]struct{}, len(input.Nicknames))
	results := make([]InviteByNicknamesResult, 0, len(input.Nicknames))

	for _, nickname := range input.Nicknames {
		key := strings.ToLower(nickname)
		if _, dup := seen[key]; dup {
			continue
		}
		seen[key] = struct{}{}

		display, resolveErr := uc.nicknameResolver.ResolveByNickname(ctx, nickname)
		if resolveErr != nil {
			results = append(results, InviteByNicknamesResult{Nickname: nickname, Status: InviteResultFailed, Reason: "nickname lookup failed"})
			continue
		}
		if display == nil {
			results = append(results, InviteByNicknamesResult{Nickname: nickname, Status: InviteResultFailed, Reason: "no user found with that nickname"})
			continue
		}

		inviteeID := shared.RestoreUserID(display.ID)
		var newInv *domainGroup.GroupInvitation

		// One transaction per nickname: this is a best-effort batch, so a
		// failure here must not roll back invitations already committed
		// for earlier items.
		txErr := uc.txManager.WithTx(ctx, func(txCtx context.Context) error {
			inv, err := issueInvitation(txCtx, uc.groupRepo, uc.invitationRepo, input.GroupID, &inviteeID, invitedBy, now)
			if err != nil {
				return err
			}
			newInv = inv
			return nil
		})

		if txErr != nil {
			results = append(results, InviteByNicknamesResult{Nickname: nickname, Status: InviteResultFailed, Reason: "failed to create invitation"})
			continue
		}

		dto := groupInvitationToDTO(newInv)
		result := InviteByNicknamesResult{Nickname: nickname, Status: InviteResultInvited, Invitation: &dto, Invitee: display}

		if err := sendInvitationEmail(ctx, uc.emailSender, uc.frontendBaseURL, g.Name().String(), newInv, display); err != nil {
			result.Status = InviteResultEmailFailed
			result.Reason = "invitation created but email delivery failed"
		}
		results = append(results, result)
	}

	return &InviteByNicknamesOutput{Results: results}, nil
}
