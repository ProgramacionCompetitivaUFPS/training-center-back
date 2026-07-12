package contest

import (
	"strings"

	domainContest "github.com/training-judge-center/backend/internal/domain/contest"
)

// FilterStandingsByProfile filters participants by country/city/institution
// (AND-combined, case-insensitive exact match, applied only for non-empty
// fields) before ranking. It reads profiles already resolved into
// cached.Profiles (populated in rebuild()) — no I/O here.
//
// INDIVIDUAL entries are checked against their own profile. TEAM entries pass
// if ANY member (per cached.TeamMembers) matches the combined filter, since a
// team has no single country/city/institution of its own.
func FilterStandingsByProfile(cached *CachedStandings, country, city, institution string) []domainContest.ParticipantStanding {
	if country == "" && city == "" && institution == "" {
		return cached.Participants
	}

	matches := func(p *ParticipantProfile) bool {
		if p == nil {
			return false
		}
		if country != "" && !strings.EqualFold(p.Country, country) {
			return false
		}
		if city != "" && !strings.EqualFold(p.City, city) {
			return false
		}
		if institution != "" && !strings.EqualFold(p.Institution, institution) {
			return false
		}
		return true
	}

	filtered := make([]domainContest.ParticipantStanding, 0, len(cached.Participants))
	for _, p := range cached.Participants {
		if p.ParticipantType == "TEAM" {
			for _, memberID := range cached.TeamMembers[p.ContestantID] {
				if matches(cached.Profiles[memberID]) {
					filtered = append(filtered, p)
					break
				}
			}
			continue
		}
		if matches(cached.Profiles[p.ContestantID]) {
			filtered = append(filtered, p)
		}
	}
	return filtered
}
