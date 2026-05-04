package user_test

import (
	"testing"

	"github.com/training-judge-center/backend/internal/domain/user"
)

func TestNewUserFilter_ValidPageAndLimit(t *testing.T) {
	tests := []struct {
		name  string
		page  int
		limit int
	}{
		{"minimum valid", 1, 1},
		{"typical values", 1, 20},
		{"max limit", 1, 100},
		{"page 2", 2, 50},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := user.NewUserFilter(nil, nil, "", "", "", user.SearchByAll, "", user.SortByCreatedAt, user.SortOrderDesc, tt.page, tt.limit)
			if err != nil {
				t.Errorf("expected no error for page=%d limit=%d, got %v", tt.page, tt.limit, err)
			}
		})
	}
}

func TestNewUserFilter_InvalidPage(t *testing.T) {
	tests := []struct {
		name string
		page int
	}{
		{"zero", 0},
		{"negative", -1},
		{"large negative", -100},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := user.NewUserFilter(nil, nil, "", "", "", user.SearchByAll, "", user.SortByCreatedAt, user.SortOrderDesc, tt.page, 20)
			if err == nil {
				t.Errorf("expected error for page=%d, got nil", tt.page)
			}
		})
	}
}

func TestNewUserFilter_InvalidLimit(t *testing.T) {
	tests := []struct {
		name  string
		limit int
	}{
		{"zero", 0},
		{"negative", -1},
		{"exceeds max", 101},
		{"way over", 1000},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := user.NewUserFilter(nil, nil, "", "", "", user.SearchByAll, "", user.SortByCreatedAt, user.SortOrderDesc, 1, tt.limit)
			if err == nil {
				t.Errorf("expected error for limit=%d, got nil", tt.limit)
			}
		})
	}
}

func TestNewSortField_Valid(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected user.SortField
	}{
		{"createdAt", "createdAt", user.SortByCreatedAt},
		{"name", "name", user.SortByName},
		{"nickname", "nickname", user.SortByNickname},
		{"email", "email", user.SortByEmail},
		{"deactivatedAt", "deactivatedAt", user.SortByDeactivatedAt},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := user.NewSortField(tt.input)
			if err != nil {
				t.Fatalf("expected no error, got %v", err)
			}
			if got != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, got)
			}
		})
	}
}

func TestNewSortField_Invalid(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{"empty string", ""},
		{"sql injection attempt", "DROP TABLE users"},
		{"unknown field", "password"},
		{"uppercase variant", "CreatedAt"},
		{"partial match", "created"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := user.NewSortField(tt.input)
			if err == nil {
				t.Errorf("expected error for input %q, got nil", tt.input)
			}
		})
	}
}

func TestNewSortOrder_Valid(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected user.SortOrder
	}{
		{"lowercase asc", "asc", user.SortOrderAsc},
		{"lowercase desc", "desc", user.SortOrderDesc},
		{"uppercase ASC", "ASC", user.SortOrderAsc},
		{"uppercase DESC", "DESC", user.SortOrderDesc},
		{"mixed case Asc", "Asc", user.SortOrderAsc},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := user.NewSortOrder(tt.input)
			if err != nil {
				t.Fatalf("expected no error, got %v", err)
			}
			if got != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, got)
			}
		})
	}
}

func TestNewSortOrder_Invalid(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{"empty string", ""},
		{"unknown order", "random"},
		{"partial match", "as"},
		{"numeric", "1"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := user.NewSortOrder(tt.input)
			if err == nil {
				t.Errorf("expected error for input %q, got nil", tt.input)
			}
		})
	}
}

func TestNewSearchField_Valid(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected user.SearchField
	}{
		{"name", "name", user.SearchByName},
		{"nickname", "nickname", user.SearchByNickname},
		{"email", "email", user.SearchByEmail},
		{"institution", "institution", user.SearchByInstitution},
		{"all", "all", user.SearchByAll},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := user.NewSearchField(tt.input)
			if err != nil {
				t.Fatalf("expected no error, got %v", err)
			}
			if got != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, got)
			}
		})
	}
}

func TestNewSearchField_Invalid(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{"empty string", ""},
		{"unknown field", "password"},
		{"sql injection attempt", "name; DROP TABLE users"},
		{"uppercase variant", "Name"},
		{"partial match", "nam"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := user.NewSearchField(tt.input)
			if err == nil {
				t.Errorf("expected error for input %q, got nil", tt.input)
			}
		})
	}
}
