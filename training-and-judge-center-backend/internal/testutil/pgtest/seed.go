//go:build integration

package pgtest

import (
	"context"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// SystemUserID is the user the migrations seed; it can author problems and own groups.
const SystemUserID = "00000000-0000-0000-0000-000000000001"

func SeedUser(t *testing.T, pool *pgxpool.Pool, id, nickname string) {
	t.Helper()
	_, err := pool.Exec(context.Background(), `
		INSERT INTO users (id, email, password, name, nickname, country, city, institution)
		VALUES ($1, $2, 'hash', $3, $4, 'Colombia', 'Bogota', 'UNAL')`,
		id, nickname+"@example.com", nickname, nickname)
	if err != nil {
		t.Fatalf("pgtest: seed user %s: %v", nickname, err)
	}
}

func SeedProblem(t *testing.T, pool *pgxpool.Pool, id, slug, status string) {
	t.Helper()
	_, err := pool.Exec(context.Background(), `
		INSERT INTO problems (id, slug, title, status, accessibility, author_id)
		VALUES ($1, $2, $3, $4, 'PUBLIC', $5)`,
		id, slug, slug, status, SystemUserID)
	if err != nil {
		t.Fatalf("pgtest: seed problem %s: %v", slug, err)
	}
}

type Submission struct {
	ID          string
	ProblemID   string
	UserID      string
	Status      string
	Language    string
	SubmittedAt time.Time
}

func SeedSubmission(t *testing.T, pool *pgxpool.Pool, s Submission) {
	t.Helper()
	if s.Status == "" {
		s.Status = "PENDING"
	}
	if s.Language == "" {
		s.Language = "cpp20"
	}
	if s.SubmittedAt.IsZero() {
		s.SubmittedAt = time.Date(2026, 8, 8, 12, 0, 0, 0, time.UTC)
	}
	_, err := pool.Exec(context.Background(), `
		INSERT INTO submissions (id, problem_id, user_id, language, status, source_code_path, submitted_at)
		VALUES ($1, $2, $3, $4, $5, 'src/' || $1, $6)`,
		s.ID, s.ProblemID, s.UserID, s.Language, s.Status, s.SubmittedAt)
	if err != nil {
		t.Fatalf("pgtest: seed submission %s: %v", s.ID, err)
	}
}
