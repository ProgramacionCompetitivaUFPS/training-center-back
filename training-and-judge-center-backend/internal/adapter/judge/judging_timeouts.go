package judge

import (
	"context"
	"time"
)

// trustedSubprocessTimeout bounds how long we wait for checker/validator
// subprocesses: compiling them (NativeCompiler), running the validator
// against an input (ValidatorRunner), or invoking a compiled checker during
// output comparison (OutputComparator). All three run code written by the
// problem setter, not a contestant — trusted, but still code that can have a
// bug (e.g. an infinite loop). Without this, a stuck subprocess would leak a
// worker slot forever instead of failing the validation/judge attempt.
const trustedSubprocessTimeout = 30 * time.Second

// isTimeoutErr reports whether cmd.Run()'s error actually came from ctx's
// deadline killing the process, rather than the subprocess exiting on its
// own. exec.CommandContext kills the process by signal when ctx is done,
// which surfaces as an ordinary *exec.ExitError — indistinguishable from a
// real exit code unless the context itself is checked too. Every caller of
// this must check it before treating an *exec.ExitError as a legitimate
// verdict (rejected input / failed compile / wrong answer), or a timed-out
// subprocess gets silently misreported as one.
func isTimeoutErr(ctx context.Context, err error) bool {
	return err != nil && ctx.Err() != nil
}
