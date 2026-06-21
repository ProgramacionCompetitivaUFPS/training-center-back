package problem

import "context"

type ProblemStats struct {
	TotalSubmissions int
	UniqueAttempted  int
	UniqueSolved     int
	ByLanguage       []LanguageStat
	ByVerdict        []VerdictStat
}

type LanguageStat struct {
	Language       string
	UsersAccepted  int
	UsersAttempted int
}

type VerdictStat struct {
	Verdict string
	Count   int
}

type StatisticsProvider interface {
	GetByProblemID(ctx context.Context, problemID string) (*ProblemStats, error)
}
