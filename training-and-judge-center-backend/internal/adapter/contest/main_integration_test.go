//go:build integration

package contest

import (
	"os"
	"testing"

	"github.com/training-judge-center/backend/internal/testutil/pgtest"
)

func TestMain(m *testing.M) {
	os.Exit(pgtest.Main(m))
}
