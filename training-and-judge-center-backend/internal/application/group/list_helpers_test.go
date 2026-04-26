package group

import (
	"testing"

	domainGroup "github.com/training-judge-center/backend/internal/domain/group"
)

func TestParseSort_EmptyReturnsDefault(t *testing.T) {
	got, err := parseSort("", domainGroup.SortByName, []domainGroup.SortField{domainGroup.SortByName, domainGroup.SortByCreatedAt})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != domainGroup.SortByName {
		t.Errorf("expected default SortByName, got %s", got)
	}
}

func TestParseOrder_DescReturnsOrderDesc(t *testing.T) {
	got, err := parseOrder("desc")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != domainGroup.OrderDesc {
		t.Errorf("expected OrderDesc, got %s", got)
	}
}
