package shared_test

import (
	"testing"

	appshared "github.com/training-judge-center/backend/internal/application/shared"
)

func TestValidatePagination(t *testing.T) {
	tests := []struct {
		name     string
		page     int
		limit    int
		maxLimit int
		wantErr  bool
	}{
		{"valid", 1, 10, 100, false},
		{"page zero", 0, 10, 100, true},
		{"page negative", -1, 10, 100, true},
		{"limit zero", 1, 0, 100, true},
		{"limit negative", 1, -1, 100, true},
		{"limit equals max", 1, 100, 100, false},
		{"limit exceeds max", 1, 101, 100, true},
		{"limit one", 1, 1, 100, false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := appshared.ValidatePagination(tc.page, tc.limit, tc.maxLimit)
			if (err != nil) != tc.wantErr {
				t.Errorf("ValidatePagination(%d, %d, %d) error = %v, wantErr %v",
					tc.page, tc.limit, tc.maxLimit, err, tc.wantErr)
			}
		})
	}
}

func TestCalcTotalPages(t *testing.T) {
	tests := []struct {
		name  string
		total int
		limit int
		want  int
	}{
		{"zero total", 0, 10, 0},
		{"exact division", 100, 10, 10},
		{"remainder", 101, 10, 11},
		{"single item", 1, 10, 1},
		{"limit zero guard", 10, 0, 0},
		{"total less than limit", 5, 10, 1},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := appshared.CalcTotalPages(tc.total, tc.limit)
			if got != tc.want {
				t.Errorf("CalcTotalPages(%d, %d) = %d, want %d", tc.total, tc.limit, got, tc.want)
			}
		})
	}
}
