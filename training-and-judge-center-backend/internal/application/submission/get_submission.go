package submission

import (
	"context"
	"time"

	appshared "github.com/training-judge-center/backend/internal/application/shared"
	domainSubmission "github.com/training-judge-center/backend/internal/domain/submission"
	"github.com/training-judge-center/backend/internal/domain/shared"
	"github.com/training-judge-center/backend/pkg/apperror"
)

type GetSubmissionInput struct {
	CurrentUser  appshared.CurrentUser
	SubmissionID string
}

type GetSubmissionOutput struct {
	ID            string
	Status        string
	Visibility    string
	SubmittedAt   time.Time
	JudgedAt      *time.Time
	Problem       ProblemDisplay
	Contest       *ContestDisplay
	SubmittedBy   UserDisplay
	Language      string
	Compiler      string
	ExecutionTime *int
	MemoryUsed    *int
	SourceCode    string
}

type GetSubmissionUseCase struct {
	submissionRepo   domainSubmission.Repository
	sourceCodeReader SourceCodeReader
	problemDisplay   ProblemDisplayProvider
	userDisplay      UserProvider
	contestDisplay   ContestProvider
	teamChecker      TeamMembershipChecker
	leadChecker      LeadChecker
}

func NewGetSubmissionUseCase(
	submissionRepo domainSubmission.Repository,
	sourceCodeReader SourceCodeReader,
	problemDisplay ProblemDisplayProvider,
	userDisplay UserProvider,
	contestDisplay ContestProvider,
	teamChecker TeamMembershipChecker,
	leadChecker LeadChecker,
) *GetSubmissionUseCase {
	return &GetSubmissionUseCase{
		submissionRepo:   submissionRepo,
		sourceCodeReader: sourceCodeReader,
		problemDisplay:   problemDisplay,
		userDisplay:      userDisplay,
		contestDisplay:   contestDisplay,
		teamChecker:      teamChecker,
		leadChecker:      leadChecker,
	}
}

func (uc *GetSubmissionUseCase) Execute(ctx context.Context, in GetSubmissionInput) (*GetSubmissionOutput, error) {
	sub, err := uc.submissionRepo.FindByID(ctx, in.SubmissionID)
	if err != nil {
		return nil, err
	}

	viewerID := in.CurrentUser.ID
	authorID := sub.UserID().String()
	isAuthor := viewerID == authorID
	isAdmin := in.CurrentUser.Role == shared.RoleAdmin

	if !isAuthor && !isAdmin && sub.Visibility().IsPrivate() {
		allowed := false
		if cid := sub.ContestID(); cid != nil {
			isLead, err := uc.leadChecker.IsLeadOfContestGroup(ctx, *cid, viewerID)
			if err != nil {
				return nil, err
			}
			if isLead {
				allowed = true
			}
			if !allowed {
				isTeam, err := uc.teamChecker.IsTeammate(ctx, *cid, viewerID, authorID)
				if err != nil {
					return nil, err
				}
				if isTeam {
					allowed = true
				}
			}
		}
		if !allowed {
			return nil, apperror.NewForbidden(domainSubmission.ErrCodeAccessDenied, "access denied")
		}
	}

	sourceCode, err := uc.sourceCodeReader.Read(ctx, sub.SourceCodePath())
	if err != nil {
		return nil, err
	}

	problem, err := uc.problemDisplay.GetProblemByID(ctx, sub.ProblemID())
	if err != nil {
		return nil, err
	}

	submittedBy, err := uc.userDisplay.GetUserByID(ctx, authorID)
	if err != nil {
		return nil, err
	}

	var contest *ContestDisplay
	if cid := sub.ContestID(); cid != nil {
		cd, err := uc.contestDisplay.GetContestByID(ctx, *cid)
		if err != nil {
			return nil, err
		}
		contest = cd
	}

	return &GetSubmissionOutput{
		ID:            sub.ID(),
		Status:        sub.Status().String(),
		Visibility:    sub.Visibility().String(),
		SubmittedAt:   sub.SubmittedAt(),
		JudgedAt:      sub.JudgedAt(),
		Problem:       *problem,
		Contest:       contest,
		SubmittedBy:   *submittedBy,
		Language:      sub.Language().String(),
		Compiler:      sub.Compiler(),
		ExecutionTime: sub.TimeMs(),
		MemoryUsed:    sub.MemoryKb(),
		SourceCode:    string(sourceCode),
	}, nil
}
