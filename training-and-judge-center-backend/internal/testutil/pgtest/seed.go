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

func SeedGroup(t *testing.T, pool *pgxpool.Pool, id, name, createdBy string) {
	t.Helper()
	_, err := pool.Exec(context.Background(), `
		INSERT INTO groups (id, name, visibility, join_policy, created_by)
		VALUES ($1, $2, 'VISIBLE', 'OPEN', $3)`,
		id, name, createdBy)
	if err != nil {
		t.Fatalf("pgtest: seed group %s: %v", name, err)
	}
}

func SeedGroupMember(t *testing.T, pool *pgxpool.Pool, memberID, groupID, userID, role string) {
	t.Helper()
	_, err := pool.Exec(context.Background(), `
		INSERT INTO group_members (id, group_id, user_id, member_role)
		VALUES ($1, $2, $3, $4)`,
		memberID, groupID, userID, role)
	if err != nil {
		t.Fatalf("pgtest: seed group member %s: %v", memberID, err)
	}
}

type Contest struct {
	ID      string
	GroupID string
	OwnerID string
	Name    string
	Start   time.Time
	End     time.Time
}

func SeedContest(t *testing.T, pool *pgxpool.Pool, c Contest) {
	t.Helper()
	if c.Name == "" {
		c.Name = "Contest " + c.ID
	}
	_, err := pool.Exec(context.Background(), `
		INSERT INTO contests (id, name, start_time, end_time, group_id, owner_id)
		VALUES ($1, $2, $3, $4, $5, $6)`,
		c.ID, c.Name, c.Start, c.End, c.GroupID, c.OwnerID)
	if err != nil {
		t.Fatalf("pgtest: seed contest %s: %v", c.ID, err)
	}
}

func SeedContestProblem(t *testing.T, pool *pgxpool.Pool, id, contestID, problemID string, order int) {
	t.Helper()
	_, err := pool.Exec(context.Background(), `
		INSERT INTO contest_problems (id, contest_id, problem_id, "order")
		VALUES ($1, $2, $3, $4)`,
		id, contestID, problemID, order)
	if err != nil {
		t.Fatalf("pgtest: seed contest problem %s: %v", id, err)
	}
}

func SeedTeam(t *testing.T, pool *pgxpool.Pool, id, name, createdBy string) {
	t.Helper()
	_, err := pool.Exec(context.Background(), `
		INSERT INTO teams (id, name, created_by, created_at)
		VALUES ($1, $2, $3, now())`,
		id, name, createdBy)
	if err != nil {
		t.Fatalf("pgtest: seed team %s: %v", name, err)
	}
}

func SeedTeamMember(t *testing.T, pool *pgxpool.Pool, id, teamID, userID string) {
	t.Helper()
	_, err := pool.Exec(context.Background(), `
		INSERT INTO team_members (id, team_id, user_id, joined_at)
		VALUES ($1, $2, $3, now())`,
		id, teamID, userID)
	if err != nil {
		t.Fatalf("pgtest: seed team member %s: %v", id, err)
	}
}
