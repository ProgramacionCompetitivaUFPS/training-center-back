package timeutil_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/training-judge-center/backend/pkg/timeutil"
)

func TestFormat_AlwaysUTCWithZSuffix(t *testing.T) {
	bogota := time.FixedZone("UTC-5", -5*60*60)
	tests := []struct {
		name string
		in   time.Time
		want string
	}{
		{"already UTC", time.Date(2026, 8, 8, 12, 30, 45, 0, time.UTC), "2026-08-08T12:30:45Z"},
		{"another zone is converted", time.Date(2026, 8, 8, 12, 30, 45, 0, bogota), "2026-08-08T17:30:45Z"},
		{"crossing midnight", time.Date(2026, 12, 31, 23, 0, 0, 0, bogota), "2027-01-01T04:00:00Z"},
		{"sub-second precision is dropped", time.Date(2026, 8, 8, 12, 30, 45, 999_000_000, time.UTC), "2026-08-08T12:30:45Z"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, timeutil.Format(tt.in))
		})
	}
}
