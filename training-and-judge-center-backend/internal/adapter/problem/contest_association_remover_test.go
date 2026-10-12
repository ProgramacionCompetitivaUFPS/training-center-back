package problem

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/training-judge-center/backend/pkg/apperror"
)

func rowsOfContests(ids ...string) *mockRows {
	rows := &mockRows{}
	for _, id := range ids {
		rows.scanFns = append(rows.scanFns, func(dest ...any) error {
			*dest[0].(*string) = id
			return nil
		})
	}
	return rows
}

func TestContestAssociationRemover_RenumbersTheContestsItLeft(t *testing.T) {
	var statements []string
	var renumberedContests []string
	q := &mockQuerier{
		queryFn: func(_ context.Context, sql string, _ ...interface{}) (pgx.Rows, error) {
			statements = append(statements, sql)
			return rowsOfContests("contest-a", "contest-b"), nil
		},
		execFn: func(_ context.Context, sql string, args ...interface{}) (pgconn.CommandTag, error) {
			statements = append(statements, sql)
			renumberedContests = args[0].([]string)
			return pgconn.NewCommandTag("UPDATE"), nil
		},
	}

	if err := NewContestAssociationRemover(q).RemoveProblemFromContests(context.Background(), "problem-1"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(statements) != 3 || !strings.HasPrefix(statements[0], "DELETE") ||
		!strings.Contains(statements[1], "+ 1000000") || !strings.Contains(statements[2], "ROW_NUMBER()") {
		t.Errorf("expected delete, shift out of range, renumber; got %v", statements)
	}
	if len(renumberedContests) != 2 || renumberedContests[0] != "contest-a" || renumberedContests[1] != "contest-b" {
		t.Errorf("renumbered contests: got %v", renumberedContests)
	}
}

func TestContestAssociationRemover_ProblemInNoContest_DoesNotRenumber(t *testing.T) {
	q := &mockQuerier{
		queryFn: func(context.Context, string, ...interface{}) (pgx.Rows, error) { return rowsOfContests(), nil },
		execFn: func(context.Context, string, ...interface{}) (pgconn.CommandTag, error) {
			t.Fatal("nothing should be renumbered when the problem was in no contest")
			return pgconn.CommandTag{}, nil
		},
	}

	if err := NewContestAssociationRemover(q).RemoveProblemFromContests(context.Background(), "problem-1"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestContestAssociationRemover_DatabaseError_ReturnsInternal(t *testing.T) {
	q := &mockQuerier{
		queryFn: func(context.Context, string, ...interface{}) (pgx.Rows, error) { return nil, errors.New("boom") },
	}

	err := NewContestAssociationRemover(q).RemoveProblemFromContests(context.Background(), "problem-1")

	assertAppErrorKind(t, err, apperror.KindInternal)
}
