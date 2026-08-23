package judge

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/moby/moby/client"
)

// The failure this exists for: Docker turns a bind source it cannot find into an
// empty directory instead of an error, so a mount pointing somewhere else would
// let every judging run and every checker compare against nothing.
func TestVerifySharedVolume_FailsWhenTheSandboxSeesSomethingElse(t *testing.T) {
	tests := []struct {
		name     string
		sandbox  string // what cat returns inside the container
		wantFail bool
	}{
		{"the sandbox reads back what the worker wrote", probeMarker, false},
		{"the sandbox sees an empty directory", "", true},
		{"the sandbox sees another volume's file", "something else", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p, _ := newTestPool(t)
			docker := &mockDockerExecClient{
				execAttachFn: func(context.Context, string, client.ExecAttachOptions) (client.ExecAttachResult, error) {
					return fakeAttach(stdcopyFrame(1, []byte(tt.sandbox))), nil
				},
			}

			err := VerifySharedVolume(context.Background(), p, docker, t.TempDir())

			if tt.wantFail && err == nil {
				t.Error("expected the probe to fail, got nil")
			}
			if !tt.wantFail && err != nil {
				t.Errorf("expected the probe to pass, got: %v", err)
			}
		})
	}
}

// The probe must not leave the directory it wrote behind, whichever way it went.
func TestVerifySharedVolume_CleansUpAfterItself(t *testing.T) {
	p, _ := newTestPool(t)
	docker := &mockDockerExecClient{
		execAttachFn: func(context.Context, string, client.ExecAttachOptions) (client.ExecAttachResult, error) {
			return fakeAttach(stdcopyFrame(1, []byte(probeMarker))), nil
		},
	}
	root := t.TempDir()

	if err := VerifySharedVolume(context.Background(), p, docker, root); err != nil {
		t.Fatalf("VerifySharedVolume: %v", err)
	}

	entries, err := os.ReadDir(root)
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range entries {
		if strings.Contains(e.Name(), "probe") {
			t.Errorf("the probe left %q behind", filepath.Join(root, e.Name()))
		}
	}
}
