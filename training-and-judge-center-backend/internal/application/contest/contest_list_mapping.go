package contest

import (
	"time"

	domainContest "github.com/training-judge-center/backend/internal/domain/contest"
)

// buildContestFilters translates the string-based status/sort/order filters
// shared by ListContestsUseCase and ListMyContestsUseCase into domain types.
// GroupID and Search are set by the caller — this only covers the fields
// both use cases parse identically.
func buildContestFilters(status *string, sortBy, order string, page, limit int) domainContest.ListFilters {
	filters := domainContest.ListFilters{
		Page:  page,
		Limit: limit,
	}

	if status != nil {
		switch *status {
		case "SCHEDULED":
			s := domainContest.StatusScheduled
			filters.Status = &s
		case "ACTIVE":
			s := domainContest.StatusActive
			filters.Status = &s
		case "FINISHED":
			s := domainContest.StatusFinished
			filters.Status = &s
		}
	}

	switch sortBy {
	case "name":
		filters.SortBy = domainContest.SortByName
	case "createdAt":
		filters.SortBy = domainContest.SortByCreatedAt
	default:
		filters.SortBy = domainContest.SortByStartTime
	}

	if order == "asc" {
		filters.Order = domainContest.OrderAsc
	} else {
		filters.Order = domainContest.OrderDesc
	}

	return filters
}

// toContestListItem maps a domain Contest plus its resolved display data into
// the shared ContestListItem output shape.
func toContestListItem(c *domainContest.Contest, now time.Time, participantCount int, isRegistered bool) ContestListItem {
	return ContestListItem{
		ID:                c.ID(),
		Name:              c.Name().Value(),
		Description:       truncateDescription(c.Description()),
		StartTime:         c.StartTime(),
		EndTime:           c.EndTime(),
		Duration:          c.Duration() * 60,
		Status:            c.Status(now).String(),
		Penalty:           c.Penalty().Value(),
		FreezeMinutes:     c.FreezeMinutes(),
		EnablePostContest: c.EnablePostContest(),
		ParticipantCount:  participantCount,
		IsRegistered:      isRegistered,
		ProblemCount:      len(c.Problems()),
	}
}
