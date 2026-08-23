package judge

import (
	"archive/tar"
	"bufio"
	"bytes"
	"context"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"

	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/moby/moby/api/types/container"
	"github.com/moby/moby/client"
	judgepool "github.com/training-judge-center/backend/internal/adapter/judge/pool"
	"github.com/training-judge-center/backend/pkg/apperror"
)

// ── mockQuerier ───────────────────────────────────────────────────────────────

type mockQuerier struct {
	execFn     func(ctx context.Context, sql string, args ...interface{}) (pgconn.CommandTag, error)
	queryFn    func(ctx context.Context, sql string, args ...interface{}) (pgx.Rows, error)
	queryRowFn func(ctx context.Context, sql string, args ...interface{}) pgx.Row
}

func (m *mockQuerier) Exec(ctx context.Context, sql string, args ...interface{}) (pgconn.CommandTag, error) {
	if m.execFn != nil {
		return m.execFn(ctx, sql, args...)
	}
	return pgconn.NewCommandTag("OK"), nil
}

func (m *mockQuerier) Query(ctx context.Context, sql string, args ...interface{}) (pgx.Rows, error) {
	if m.queryFn != nil {
		return m.queryFn(ctx, sql, args...)
	}
	return &mockRows{}, nil
}

func (m *mockQuerier) QueryRow(ctx context.Context, sql string, args ...interface{}) pgx.Row {
	if m.queryRowFn != nil {
		return m.queryRowFn(ctx, sql, args...)
	}
	return &mockRow{scanFn: func(dest ...any) error { return nil }}
}

// ── mockRow ───────────────────────────────────────────────────────────────────

type mockRow struct {
	scanFn func(dest ...any) error
}

func (m *mockRow) Scan(dest ...any) error {
	if m.scanFn != nil {
		return m.scanFn(dest...)
	}
	return nil
}

// ── mockRows ──────────────────────────────────────────────────────────────────

type mockRows struct {
	scanFns []func(dest ...any) error
	idx     int
	err     error
}

func (m *mockRows) Next() bool                                   { m.idx++; return m.idx <= len(m.scanFns) }
func (m *mockRows) Close()                                       {}
func (m *mockRows) Err() error                                   { return m.err }
func (m *mockRows) CommandTag() pgconn.CommandTag                { return pgconn.CommandTag{} }
func (m *mockRows) FieldDescriptions() []pgconn.FieldDescription { return nil }
func (m *mockRows) Values() ([]any, error)                       { return nil, nil }
func (m *mockRows) RawValues() [][]byte                          { return nil }
func (m *mockRows) Conn() *pgx.Conn                              { return nil }
func (m *mockRows) Scan(dest ...any) error {
	if m.idx < 1 || m.idx > len(m.scanFns) {
		return errors.New("scan called out of bounds")
	}
	return m.scanFns[m.idx-1](dest...)
}

// ── mockGCSReader ─────────────────────────────────────────────────────────────

type mockGCSReader struct {
	readObjectFn func(ctx context.Context, object string) (io.ReadCloser, error)
}

func (m *mockGCSReader) readObject(ctx context.Context, object string) (io.ReadCloser, error) {
	if m.readObjectFn != nil {
		return m.readObjectFn(ctx, object)
	}
	return io.NopCloser(bytes.NewReader(nil)), nil
}

// ── test helpers ──────────────────────────────────────────────────────────────

func assertAppErrorKind(t *testing.T, err error, want apperror.Kind) {
	t.Helper()
	var appErr *apperror.AppError
	if !errors.As(err, &appErr) {
		t.Fatalf("expected *apperror.AppError, got %T: %v", err, err)
	}
	if appErr.Kind != want {
		t.Errorf("expected kind %q, got %q", want, appErr.Kind)
	}
}

// ── pipeConn: net.Conn over an io.PipeReader, for fake attach results ───────

type pipeConn struct{ *io.PipeReader }

func (pipeConn) Write(b []byte) (int, error)        { return len(b), nil }
func (pipeConn) LocalAddr() net.Addr                { return nil }
func (pipeConn) RemoteAddr() net.Addr               { return nil }
func (pipeConn) SetDeadline(_ time.Time) error      { return nil }
func (pipeConn) SetReadDeadline(_ time.Time) error  { return nil }
func (pipeConn) SetWriteDeadline(_ time.Time) error { return nil }

// ── attach helpers ──────────────────────────────────────────────────────────

// fakeAttach returns an ExecAttachResult whose reader yields content then EOF.
func fakeAttach(content []byte) client.ExecAttachResult {
	pr, pw := io.Pipe()
	go func() {
		_, _ = pw.Write(content)
		pw.Close()
	}()
	conn := pipeConn{pr}
	return client.ExecAttachResult{
		HijackedResponse: client.HijackedResponse{
			Conn:   conn,
			Reader: bufio.NewReader(conn),
		},
	}
}

// blockingAttach returns an ExecAttachResult that blocks until ctx is done.
func blockingAttach(ctx context.Context) client.ExecAttachResult {
	pr, pw := io.Pipe()
	go func() {
		<-ctx.Done()
		pw.CloseWithError(ctx.Err())
	}()
	conn := pipeConn{pr}
	return client.ExecAttachResult{
		HijackedResponse: client.HijackedResponse{
			Conn:   conn,
			Reader: bufio.NewReader(conn),
		},
	}
}

// stdcopyFrame encodes data in Docker's multiplexed stream format.
// streamType: 1 = stdout, 2 = stderr.
func stdcopyFrame(streamType byte, data []byte) []byte {
	frame := make([]byte, 8+len(data))
	frame[0] = streamType
	binary.BigEndian.PutUint32(frame[4:], uint32(len(data)))
	copy(frame[8:], data)
	return frame
}

// outputTar wraps content in a tar archive, mimicking CopyFromContainer's response.
func outputTar(content []byte) io.ReadCloser {
	var buf bytes.Buffer
	tw := tar.NewWriter(&buf)
	_ = tw.WriteHeader(&tar.Header{Name: "output.txt", Mode: 0644, Size: int64(len(content))})
	_, _ = tw.Write(content)
	_ = tw.Close()
	return io.NopCloser(&buf)
}

// statsBody returns JSON-encoded StatsResponse with the given peak memory usage.
func statsBody(maxUsageBytes uint64) io.ReadCloser {
	stats := container.StatsResponse{
		MemoryStats: container.MemoryStats{MaxUsage: maxUsageBytes},
	}
	data, _ := json.Marshal(stats)
	return io.NopCloser(bytes.NewReader(data))
}

// ── mockDockerExecClient ────────────────────────────────────────────────────

type mockDockerExecClient struct {
	copyToContainerFn   func(context.Context, string, client.CopyToContainerOptions) (client.CopyToContainerResult, error)
	execCreateFn        func(context.Context, string, client.ExecCreateOptions) (client.ExecCreateResult, error)
	execAttachFn        func(context.Context, string, client.ExecAttachOptions) (client.ExecAttachResult, error)
	execInspectFn       func(context.Context, string, client.ExecInspectOptions) (client.ExecInspectResult, error)
	copyFromContainerFn func(context.Context, string, client.CopyFromContainerOptions) (client.CopyFromContainerResult, error)
	containerStatsFn    func(context.Context, string, client.ContainerStatsOptions) (client.ContainerStatsResult, error)

	execCreateCnt atomic.Int64
}

func (m *mockDockerExecClient) CopyToContainer(ctx context.Context, id string, opts client.CopyToContainerOptions) (client.CopyToContainerResult, error) {
	if m.copyToContainerFn != nil {
		return m.copyToContainerFn(ctx, id, opts)
	}
	return client.CopyToContainerResult{}, nil
}

func (m *mockDockerExecClient) ExecCreate(ctx context.Context, id string, opts client.ExecCreateOptions) (client.ExecCreateResult, error) {
	m.execCreateCnt.Add(1)
	if m.execCreateFn != nil {
		return m.execCreateFn(ctx, id, opts)
	}
	return client.ExecCreateResult{ID: "exec-1"}, nil
}

func (m *mockDockerExecClient) ExecAttach(ctx context.Context, id string, opts client.ExecAttachOptions) (client.ExecAttachResult, error) {
	if m.execAttachFn != nil {
		return m.execAttachFn(ctx, id, opts)
	}
	return fakeAttach(nil), nil
}

func (m *mockDockerExecClient) ExecInspect(ctx context.Context, id string, opts client.ExecInspectOptions) (client.ExecInspectResult, error) {
	if m.execInspectFn != nil {
		return m.execInspectFn(ctx, id, opts)
	}
	return client.ExecInspectResult{ExitCode: 0}, nil
}

func (m *mockDockerExecClient) CopyFromContainer(ctx context.Context, id string, opts client.CopyFromContainerOptions) (client.CopyFromContainerResult, error) {
	if m.copyFromContainerFn != nil {
		return m.copyFromContainerFn(ctx, id, opts)
	}
	return client.CopyFromContainerResult{Content: outputTar(nil)}, nil
}

func (m *mockDockerExecClient) ContainerStats(ctx context.Context, id string, opts client.ContainerStatsOptions) (client.ContainerStatsResult, error) {
	if m.containerStatsFn != nil {
		return m.containerStatsFn(ctx, id, opts)
	}
	return client.ContainerStatsResult{Body: statsBody(0)}, nil
}

// ── mockPoolDockerClient: satisfies pool's unexported dockerLifecycle ───────

type mockPoolDockerClient struct {
	idCounter atomic.Int64
	// lastCreateMemory is how a test sees the ceiling a claim asked for: the
	// pool creates the container straight at it.
	lastCreateMemory atomic.Int64
}

func (m *mockPoolDockerClient) ContainerCreate(_ context.Context, opts client.ContainerCreateOptions) (client.ContainerCreateResult, error) {
	if opts.HostConfig != nil {
		m.lastCreateMemory.Store(opts.HostConfig.Resources.Memory)
	}
	return client.ContainerCreateResult{ID: fmt.Sprintf("pool-ctr-%d", m.idCounter.Add(1))}, nil
}

func (m *mockPoolDockerClient) ContainerUpdate(_ context.Context, _ string, _ client.ContainerUpdateOptions) (client.ContainerUpdateResult, error) {
	return client.ContainerUpdateResult{}, nil
}

func (m *mockPoolDockerClient) ContainerStart(_ context.Context, _ string, _ client.ContainerStartOptions) (client.ContainerStartResult, error) {
	return client.ContainerStartResult{}, nil
}

func (m *mockPoolDockerClient) ContainerRemove(_ context.Context, _ string, _ client.ContainerRemoveOptions) (client.ContainerRemoveResult, error) {
	return client.ContainerRemoveResult{}, nil
}

func (m *mockPoolDockerClient) Ping(_ context.Context, _ client.PingOptions) (client.PingResult, error) {
	return client.PingResult{}, nil
}

// ── pool test setup ─────────────────────────────────────────────────────────

const (
	testLang     = "cpp20"
	testMemBytes = 512 * 1024 * 1024 // 512 MB
	// testProblemMemoryKb is a problem limit well under testMemBytes, so tests
	// can tell the problem's ceiling from the pool's.
	testProblemMemoryKb = 128 * 1024 // 128 MB
)

func testPoolCfg() judgepool.PoolConfig {
	return judgepool.PoolConfig{
		BudgetBytes:  4 * testMemBytes,
		IdleTimeout:  time.Hour,
		ReapInterval: time.Hour,
		Languages: map[string]judgepool.LanguageConfig{
			testLang: {Image: "judge:cpp20", MemoryBytes: testMemBytes},
		},
	}
}

func newTestPool(t *testing.T) (*judgepool.Pool, *mockPoolDockerClient) {
	t.Helper()
	poolMock := &mockPoolDockerClient{}
	p := judgepool.NewPool(testPoolCfg(), poolMock)
	p.Start()
	t.Cleanup(p.Stop)
	return p, poolMock
}

// recordExecs numbers the exec calls so the attach mock can tell the compile
// (the first) from the cleanup (the second), and keeps every command issued.
func recordExecs(docker *mockDockerExecClient, cmds *[][]string) {
	docker.execCreateFn = func(_ context.Context, _ string, opts client.ExecCreateOptions) (client.ExecCreateResult, error) {
		*cmds = append(*cmds, opts.Cmd)
		return client.ExecCreateResult{ID: fmt.Sprintf("exec-%d", len(*cmds))}, nil
	}
}

// ── artifact test fixtures ──────────────────────────────────────────────────

// testArtifactCfg is one language's artifact config, with the {name} token the
// compiler and both light pool sessions substitute for their role.
func testArtifactCfg() ArtifactConfig {
	return ArtifactConfig{
		Languages: map[string]ArtifactLanguageConfig{
			testLang: {
				SourcePath:   "/sandbox/{name}.cpp",
				CompileCmd:   "g++ -o /sandbox/{name} /sandbox/{name}.cpp",
				ArtifactPath: "/sandbox/{name}",
				RunCmd:       "/sandbox/{name}",
			},
		},
	}
}

func storedArtifact(content string) *mockGCSReader {
	return &mockGCSReader{
		readObjectFn: func(_ context.Context, _ string) (io.ReadCloser, error) {
			return io.NopCloser(strings.NewReader(content)), nil
		},
	}
}

// firstTarEntry reads back what a test sent through CopyToContainer.
func firstTarEntry(t *testing.T, r io.Reader) (string, []byte) {
	t.Helper()
	name, _, data := firstTarEntryMode(t, r)
	return name, data
}

// firstTarEntryMode adds the mode, which is what decides whether the sandbox can
// execute the artifact at all.
func firstTarEntryMode(t *testing.T, r io.Reader) (string, int64, []byte) {
	t.Helper()
	tr := tar.NewReader(r)
	hdr, err := tr.Next()
	if err != nil {
		t.Fatalf("could not read the tar sent to the container: %v", err)
	}
	data, _ := io.ReadAll(tr)
	return hdr.Name, hdr.Mode, data
}

// newTestJudgingDir lays out one judging's slot under a temp root. It does not
// call createJudgingDir: that one hands output.txt to the sandbox user and
// leaves the directory unwritable, and both of those need root.
func newTestJudgingDir(t *testing.T) string {
	t.Helper()
	dir := filepath.Join(t.TempDir(), "judging")
	if err := os.Mkdir(dir, 0o755); err != nil {
		t.Fatalf("creating the judging directory: %v", err)
	}
	if err := os.WriteFile(judgingOutputPath(dir), nil, judgingFileMode); err != nil {
		t.Fatalf("creating output.txt: %v", err)
	}
	return dir
}

// writeContestantOutput stands in for the sandbox: it leaves in the judging
// directory what the contestant's program would have printed.
func writeContestantOutput(t *testing.T, judgingDir string, output []byte) {
	t.Helper()
	if err := os.WriteFile(judgingOutputPath(judgingDir), output, judgingFileMode); err != nil {
		t.Fatalf("writing the contestant's output: %v", err)
	}
}
