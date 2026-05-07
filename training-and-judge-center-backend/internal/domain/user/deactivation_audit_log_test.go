package user_test

import (
	"testing"

	"github.com/training-judge-center/backend/internal/domain/user"
)

func TestNewDeactivationAuditLog_EmptyID_ReturnsError(t *testing.T) {
	_, err := user.NewDeactivationAuditLog("", "user-id", "email@example.com", "nickname", testNow, nil, nil)
	if err == nil {
		t.Error("expected error for empty id, got nil")
	}
}

func TestNewDeactivationAuditLog_SetsFields(t *testing.T) {
	ip := "192.168.1.1"
	ua := "Mozilla/5.0"

	log, err := user.NewDeactivationAuditLog("log-id", "user-id", "email@example.com", "nickname", testNow, &ip, &ua)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if log.ID() != "log-id" {
		t.Errorf("ID: got %q, want %q", log.ID(), "log-id")
	}
	if log.UserID() != "user-id" {
		t.Errorf("UserID: got %q, want %q", log.UserID(), "user-id")
	}
	if log.OriginalEmail() != "email@example.com" {
		t.Errorf("OriginalEmail: got %q, want %q", log.OriginalEmail(), "email@example.com")
	}
	if log.OriginalNickname() != "nickname" {
		t.Errorf("OriginalNickname: got %q, want %q", log.OriginalNickname(), "nickname")
	}
	if !log.OccurredAt().Equal(testNow) {
		t.Errorf("OccurredAt: got %v, want %v", log.OccurredAt(), testNow)
	}
	if log.IP() == nil || *log.IP() != ip {
		t.Errorf("IP: got %v, want %q", log.IP(), ip)
	}
	if log.UserAgent() == nil || *log.UserAgent() != ua {
		t.Errorf("UserAgent: got %v, want %q", log.UserAgent(), ua)
	}
}

func TestNewDeactivationAuditLog_NilOptionals_Stored(t *testing.T) {
	log, err := user.NewDeactivationAuditLog("log-id", "user-id", "email@example.com", "nickname", testNow, nil, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if log.IP() != nil {
		t.Errorf("IP: got %v, want nil", log.IP())
	}
	if log.UserAgent() != nil {
		t.Errorf("UserAgent: got %v, want nil", log.UserAgent())
	}
}
