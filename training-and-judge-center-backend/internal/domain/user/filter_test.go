package user

import "testing"

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
			_, err := NewUserFilter(nil, nil, "", "", "", SearchByAll, "", SortByCreatedAt, SortOrderDesc, tt.page, tt.limit)
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
			_, err := NewUserFilter(nil, nil, "", "", "", SearchByAll, "", SortByCreatedAt, SortOrderDesc, tt.page, 20)
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
			_, err := NewUserFilter(nil, nil, "", "", "", SearchByAll, "", SortByCreatedAt, SortOrderDesc, 1, tt.limit)
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
		expected SortField
	}{
		{"createdAt", "createdAt", SortByCreatedAt},
		{"name", "name", SortByName},
		{"nickname", "nickname", SortByNickname},
		{"email", "email", SortByEmail},
		{"deactivatedAt", "deactivatedAt", SortByDeactivatedAt},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NewSortField(tt.input)
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
			_, err := NewSortField(tt.input)
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
		expected SortOrder
	}{
		{"lowercase asc", "asc", SortOrderAsc},
		{"lowercase desc", "desc", SortOrderDesc},
		{"uppercase ASC", "ASC", SortOrderAsc},
		{"uppercase DESC", "DESC", SortOrderDesc},
		{"mixed case Asc", "Asc", SortOrderAsc},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NewSortOrder(tt.input)
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
			_, err := NewSortOrder(tt.input)
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
		expected SearchField
	}{
		{"name", "name", SearchByName},
		{"nickname", "nickname", SearchByNickname},
		{"email", "email", SearchByEmail},
		{"institution", "institution", SearchByInstitution},
		{"all", "all", SearchByAll},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NewSearchField(tt.input)
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
			_, err := NewSearchField(tt.input)
			if err == nil {
				t.Errorf("expected error for input %q, got nil", tt.input)
			}
		})
	}
}
