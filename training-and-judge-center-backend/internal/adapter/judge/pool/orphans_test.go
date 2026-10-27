package pool

import (
	"context"
	"errors"
	"testing"

	"github.com/moby/moby/api/types/container"
	"github.com/moby/moby/client"
)

type mockCleaner struct {
	listed     []container.Summary
	listErr    error
	removeErrs map[string]error
	gotFilters client.Filters
	gotAll     bool
	removed    []string
}

func (m *mockCleaner) ContainerList(_ context.Context, opts client.ContainerListOptions) (client.ContainerListResult, error) {
	m.gotFilters = opts.Filters
	m.gotAll = opts.All
	return client.ContainerListResult{Items: m.listed}, m.listErr
}

func (m *mockCleaner) ContainerRemove(_ context.Context, id string, opts client.ContainerRemoveOptions) (client.ContainerRemoveResult, error) {
	if !opts.Force {
		return client.ContainerRemoveResult{}, errors.New("removal must be forced: the sandboxes are running")
	}
	if err := m.removeErrs[id]; err != nil {
		return client.ContainerRemoveResult{}, err
	}
	m.removed = append(m.removed, id)
	return client.ContainerRemoveResult{}, nil
}

func TestRemoveOrphans_ListsOnlyByTheSandboxLabel(t *testing.T) {
	m := &mockCleaner{listed: []container.Summary{{ID: "a"}, {ID: "b"}}}

	n, err := RemoveOrphans(context.Background(), m)
	if err != nil {
		t.Fatalf("RemoveOrphans: %v", err)
	}
	if n != 2 || len(m.removed) != 2 {
		t.Errorf("removed %d (%v), want 2", n, m.removed)
	}
	want := SandboxLabelKey + "=" + SandboxLabelValue
	if len(m.gotFilters) != 1 || !m.gotFilters["label"][want] {
		t.Errorf("filters = %v, want only label=%s", m.gotFilters, want)
	}
	if !m.gotAll {
		t.Error("stopped leftovers are orphans too: the listing must include them")
	}
}

func TestRemoveOrphans_OneFailureDoesNotStopTheRest(t *testing.T) {
	m := &mockCleaner{
		listed:     []container.Summary{{ID: "a"}, {ID: "b"}, {ID: "c"}},
		removeErrs: map[string]error{"b": errors.New("busy")},
	}

	n, err := RemoveOrphans(context.Background(), m)
	if err != nil {
		t.Fatalf("RemoveOrphans: %v", err)
	}
	if n != 2 {
		t.Errorf("removed %d, want 2", n)
	}
}

func TestRemoveOrphans_ListFailureIsAnError(t *testing.T) {
	m := &mockCleaner{listErr: errors.New("daemon down")}
	if _, err := RemoveOrphans(context.Background(), m); err == nil {
		t.Fatal("expected an error")
	}
}

func TestPool_CreatesContainersWithTheSandboxLabel(t *testing.T) {
	var got map[string]string
	m := &mockDockerClient{
		createFn: func(_ context.Context, opts client.ContainerCreateOptions) (client.ContainerCreateResult, error) {
			got = opts.Config.Labels
			return client.ContainerCreateResult{ID: "c1"}, nil
		},
	}
	p := NewPool(testCfg(2), m)
	p.Start()
	defer p.Stop()

	if _, err := p.Claim(context.Background(), "cpp20", LanguageCeiling); err != nil {
		t.Fatalf("Claim: %v", err)
	}
	if got[SandboxLabelKey] != SandboxLabelValue {
		t.Errorf("labels = %v, want %s=%s", got, SandboxLabelKey, SandboxLabelValue)
	}
}

// A graceful stop must not leave sandboxes behind, idle or busy.
func TestPool_StopRemovesEveryContainerItHolds(t *testing.T) {
	m := &mockDockerClient{}
	p := NewPool(testCfg(3), m)
	p.Start()

	busy, err := p.Claim(context.Background(), "cpp20", LanguageCeiling)
	if err != nil {
		t.Fatalf("Claim: %v", err)
	}
	idle, err := p.Claim(context.Background(), "java17", LanguageCeiling)
	if err != nil {
		t.Fatalf("Claim: %v", err)
	}
	p.Release(idle)
	_ = busy

	p.Stop()

	if got := m.removeCnt.Load(); got != 2 {
		t.Errorf("ContainerRemove calls = %d, want 2", got)
	}
	p.Stop() // idempotent: no second round of removals
	if got := m.removeCnt.Load(); got != 2 {
		t.Errorf("after a second Stop, ContainerRemove calls = %d, want 2", got)
	}
}
