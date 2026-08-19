package judge

import (
	"context"
	"time"
)

// trustedSubprocessTimeout bounds how long we wait for the checker/validator
// subprocesses still running natively: ValidatorRunner and OutputComparator.
// Without it a stuck subprocess would leak a worker slot forever instead of
// failing the validation attempt.
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
