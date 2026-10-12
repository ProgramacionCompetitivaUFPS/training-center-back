package judge

import "github.com/training-judge-center/backend/internal/domain/submission"

// JudgingCandidate is what a published problem would be judged with after a live update; empty fields keep the problem's own.
type JudgingCandidate struct {
	TestCasesKey string
	Checker      *CandidateChecker
}

// CandidateChecker compiles to a key apart from the live checker's, so building it cannot disturb a judging in progress.
type CandidateChecker struct {
	FileKey     string
	Language    submission.Language
	CompiledKey string
}
