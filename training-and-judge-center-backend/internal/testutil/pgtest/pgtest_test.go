//go:build integration

package pgtest_test

import (
	"context"
	"os"
	"testing"

	"github.com/training-judge-center/backend/internal/testutil/pgtest"
)

func TestMain(m *testing.M) {
	os.Exit(pgtest.Main(m))
}

func TestNewDB_ReturnsAMigratedDatabase(t *testing.T) {
	db := pgtest.NewDB(t)

	var hasUsers bool
	err := db.QueryRow(context.Background(), `SELECT to_regclass('users') IS NOT NULL`).Scan(&hasUsers)

	if err != nil {
		t.Fatalf("query: %v", err)
	}
	if !hasUsers {
		t.Error("table users does not exist in the new database")
	}
}

func TestNewDB_GivesEachCallItsOwnDatabase(t *testing.T) {
	ctx := context.Background()
	first := pgtest.NewDB(t)
	second := pgtest.NewDB(t)

	if _, err := first.Exec(ctx, `CREATE TABLE marker (id int)`); err != nil {
		t.Fatalf("create marker: %v", err)
	}
	var markerInSecond bool
	err := second.QueryRow(ctx, `SELECT to_regclass('marker') IS NOT NULL`).Scan(&markerInSecond)

	if err != nil {
		t.Fatalf("query: %v", err)
	}
	if markerInSecond {
		t.Error("a table created through the first database is visible in the second")
	}
}

func TestNewDB_DropsTheDatabaseWhenTheTestEnds(t *testing.T) {
	ctx := context.Background()
	observer := pgtest.NewDB(t)
	var name string
	t.Run("creates", func(t *testing.T) {
		db := pgtest.NewDB(t)
		if err := db.QueryRow(ctx, `SELECT current_database()`).Scan(&name); err != nil {
			t.Fatalf("current_database: %v", err)
		}
	})

	var stillExists bool
	err := observer.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM pg_database WHERE datname = $1)`, name).Scan(&stillExists)

	if err != nil {
		t.Fatalf("query: %v", err)
	}
	if stillExists {
		t.Errorf("database %s was not dropped after its test ended", name)
	}
}

func TestNewDB_SupportsParallelTests(t *testing.T) {
	for _, name := range []string{"a", "b", "c", "d"} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			db := pgtest.NewDB(t)

			var one int
			if err := db.QueryRow(context.Background(), `SELECT 1`).Scan(&one); err != nil || one != 1 {
				t.Errorf("select 1 = %d, err = %v", one, err)
			}
		})
	}
}
