package group

import (
	"context"
	"errors"
	"testing"
)

type mockSoleLeadGroupsProvider struct {
	groups []SoleLeadGroup
	err    error
	asked  string
}

func (m *mockSoleLeadGroupsProvider) ListSoleLeadGroups(_ context.Context, userID string) ([]SoleLeadGroup, error) {
	m.asked = userID
	return m.groups, m.err
}

func TestListSoleLeadGroups_ReturnsWhatTheProviderFinds(t *testing.T) {
	p := &mockSoleLeadGroupsProvider{groups: []SoleLeadGroup{{ID: "g1", Name: "Training"}}}
	out, err := NewListSoleLeadGroupsUseCase(p).Execute(context.Background(), "user-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if p.asked != "user-1" || len(out) != 1 || out[0].Name != "Training" {
		t.Errorf("asked=%q out=%+v", p.asked, out)
	}
}

func TestListSoleLeadGroups_PropagatesTheProviderError(t *testing.T) {
	boom := errors.New("db down")
	_, err := NewListSoleLeadGroupsUseCase(&mockSoleLeadGroupsProvider{err: boom}).Execute(context.Background(), "user-1")
	if !errors.Is(err, boom) {
		t.Errorf("expected the provider error, got %v", err)
	}
}
