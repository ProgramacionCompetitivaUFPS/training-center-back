package judge

import (
	"archive/tar"
	"bytes"
	"context"
	"io"

	"github.com/moby/moby/client"
)

// File modes for what gets copied into a container. An artifact has to be
// executable: a C++ artifact is an ELF binary the sandbox runs directly.
const (
	modeSource     int64 = 0644
	modeExecutable int64 = 0755
)

func buildTar(filename string, content []byte, mode int64) io.Reader {
	var buf bytes.Buffer
	tw := tar.NewWriter(&buf)
	_ = tw.WriteHeader(&tar.Header{Name: filename, Mode: mode, Size: int64(len(content))})
	_, _ = tw.Write(content)
	_ = tw.Close()
	return &buf
}

func extractFirstFile(r io.Reader, maxBytes int64) []byte {
	tr := tar.NewReader(r)
	if _, err := tr.Next(); err != nil {
		return nil
	}
	data, _ := io.ReadAll(&io.LimitedReader{R: tr, N: maxBytes})
	return data
}

// runAndWait runs cmd to completion inside the container, discarding whatever
// it prints. Callers use it for cleanup, where the output carries nothing.
func runAndWait(ctx context.Context, docker dockerExecClient, containerID string, cmd []string) error {
	execRes, err := docker.ExecCreate(ctx, containerID, client.ExecCreateOptions{Cmd: cmd})
	if err != nil {
		return err
	}
	att, err := docker.ExecAttach(ctx, execRes.ID, client.ExecAttachOptions{})
	if err != nil {
		return err
	}
	_, _ = io.Copy(io.Discard, att.Reader)
	att.Conn.Close()
	return nil
}
