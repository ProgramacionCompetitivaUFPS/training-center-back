package contest

import domainContest "github.com/training-judge-center/backend/internal/domain/contest"

func parseProblemColors(inputs []ProblemOrderInput) (map[string]*domainContest.ProblemColor, error) {
	changes := make(map[string]*domainContest.ProblemColor)
	for _, in := range deduplicateBySlug(inputs) {
		if in.Color == nil {
			continue
		}
		if *in.Color == nil {
			changes[in.Slug] = nil
			continue
		}
		color, err := domainContest.NewProblemColor(**in.Color)
		if err != nil {
			return nil, err
		}
		changes[in.Slug] = &color
	}
	return changes, nil
}

func colorString(c *domainContest.ProblemColor) *string {
	if c == nil {
		return nil
	}
	v := c.Value()
	return &v
}
