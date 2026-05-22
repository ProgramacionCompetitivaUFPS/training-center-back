package user_test

import (
	"errors"
	"testing"

	"github.com/training-judge-center/backend/internal/domain/shared"
	"github.com/training-judge-center/backend/internal/domain/user"
	"github.com/training-judge-center/backend/pkg/apperror"
)

func TestNewUser_Valid(t *testing.T) {
	email, _ := user.NewEmail("test@example.com")
	password, _ := user.NewPassword("SecurePass1!")
	nickname, _ := user.NewNickname("user-test")
	u, err := user.NewUser("user-id", testNow, email, password, "John Doe", nickname, "Colombia", "Bogotá", "UFPS")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if u.ID() != "user-id" {
		t.Errorf("ID = %q, want user-id", u.ID())
	}
	if u.Name() != "John Doe" {
		t.Errorf("Name = %q, want John Doe", u.Name())
	}
	if u.Email().String() != "test@example.com" {
		t.Errorf("Email = %q, want test@example.com", u.Email().String())
	}
	if u.Role() != shared.RoleContestant {
		t.Errorf("Role = %v, want RoleContestant", u.Role())
	}
	if u.Status() != user.StatusActive {
		t.Errorf("Status = %v, want StatusActive", u.Status())
	}
	if u.UpdatedAt() != nil {
		t.Error("UpdatedAt should be nil on new user")
	}
}

func TestNewUser_RequiredFieldsMissing(t *testing.T) {
	email, _ := user.NewEmail("test@example.com")
	password, _ := user.NewPassword("SecurePass1!")
	nickname, _ := user.NewNickname("user-test")
	tests := []struct {
		name                                     string
		id, userName, country, city, institution string
		wantErrCode                              string
	}{
		{"empty id", "", "John", "CO", "Bogotá", "UFPS", apperror.ErrCodeInternalError},
		{"empty name", "id", "", "CO", "Bogotá", "UFPS", apperror.ErrCodeValidationError},
		{"empty country", "id", "John", "", "Bogotá", "UFPS", apperror.ErrCodeValidationError},
		{"empty city", "id", "John", "CO", "", "UFPS", apperror.ErrCodeValidationError},
		{"empty institution", "id", "John", "CO", "Bogotá", "", apperror.ErrCodeValidationError},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := user.NewUser(tt.id, testNow, email, password, tt.userName, nickname, tt.country, tt.city, tt.institution)
			if err == nil {
				t.Fatal("expected error, got nil")
			}
			var appErr *apperror.AppError
			if !errors.As(err, &appErr) || appErr.Code != tt.wantErrCode {
				t.Errorf("expected error code %q, got %v", tt.wantErrCode, err)
			}
		})
	}
}

func TestUser_Update_ChangesName(t *testing.T) {
	u := makeTestUser(t)
	newName := "Jane Doe"
	if err := u.Update(&newName, nil, nil, nil, nil, testNow); err != nil {
		t.Fatalf("Update failed: %v", err)
	}
	if u.Name() != "Jane Doe" {
		t.Errorf("Name = %q, want Jane Doe", u.Name())
	}
	if u.UpdatedAt() == nil {
		t.Error("UpdatedAt should be set after update")
	}
}

func TestUser_Update_EmptyNameReturnsError(t *testing.T) {
	u := makeTestUser(t)
	empty := ""
	err := u.Update(&empty, nil, nil, nil, nil, testNow)
	var appErr *apperror.AppError
	if !errors.As(err, &appErr) || appErr.Code != apperror.ErrCodeValidationError {
		t.Errorf("expected VALIDATION_ERROR, got %v", err)
	}
}

func TestUser_Update_NilFieldsAreNoop(t *testing.T) {
	u := makeTestUser(t)
	originalName := u.Name()
	if err := u.Update(nil, nil, nil, nil, nil, testNow); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if u.Name() != originalName {
		t.Error("name should not change when nil is passed")
	}
}

func TestUser_AdminUpdate_CannotAssignAdminRole(t *testing.T) {
	u := makeTestUser(t)
	adminRole := shared.RoleAdmin
	err := u.AdminUpdate(nil, nil, nil, nil, nil, nil, &adminRole, testNow)
	var appErr *apperror.AppError
	if !errors.As(err, &appErr) || appErr.Code != user.ErrCodeCannotAssignAdminRole {
		t.Errorf("expected CANNOT_ASSIGN_ADMIN_ROLE, got %v", err)
	}
}

func TestUser_AdminUpdate_ChangesEmail(t *testing.T) {
	u := makeTestUser(t)
	newEmail, _ := user.NewEmail("new@example.com")
	if err := u.AdminUpdate(nil, nil, nil, nil, nil, &newEmail, nil, testNow); err != nil {
		t.Fatalf("AdminUpdate failed: %v", err)
	}
	if u.Email().String() != "new@example.com" {
		t.Errorf("Email = %q, want new@example.com", u.Email().String())
	}
}

func TestUser_UpdatePassword_FailsForDeactivatedUser(t *testing.T) {
	u := makeTestUser(t)
	if err := u.Deactivate("suffix123", testNow); err != nil {
		t.Fatalf("Deactivate failed: %v", err)
	}
	newPass, _ := user.NewPassword("NewPass1!")
	err := u.UpdatePassword(newPass, testNow)
	var appErr *apperror.AppError
	if !errors.As(err, &appErr) || appErr.Code != user.ErrCodeCannotUpdateDeactivated {
		t.Errorf("expected CANNOT_UPDATE_DEACTIVATED, got %v", err)
	}
}

func TestUser_UpdatePassword_Success(t *testing.T) {
	u := makeTestUser(t)
	newPass, _ := user.NewPassword("NewPass1!")
	if err := u.UpdatePassword(newPass, testNow); err != nil {
		t.Fatalf("UpdatePassword failed: %v", err)
	}
	if u.UpdatedAt() == nil {
		t.Error("UpdatedAt should be set after password update")
	}
}

func TestUser_UpdateEmail_FailsForDeactivatedUser(t *testing.T) {
	u := makeTestUser(t)
	if err := u.Deactivate("suffix456", testNow); err != nil {
		t.Fatalf("Deactivate failed: %v", err)
	}
	newEmail, _ := user.NewEmail("new@example.com")
	err := u.UpdateEmail(newEmail, testNow)
	var appErr *apperror.AppError
	if !errors.As(err, &appErr) || appErr.Code != user.ErrCodeCannotUpdateDeactivated {
		t.Errorf("expected CANNOT_UPDATE_DEACTIVATED, got %v", err)
	}
}

func TestUser_Deactivate_AnonymizesNickname(t *testing.T) {
	u := makeTestUser(t)
	originalNickname := u.Nickname().String()
	if err := u.Deactivate("anom789", testNow); err != nil {
		t.Fatalf("Deactivate failed: %v", err)
	}
	if u.Status() != user.StatusDeactivated {
		t.Errorf("Status = %v, want StatusDeactivated", u.Status())
	}
	if u.Nickname().String() == originalNickname {
		t.Error("nickname should be anonymized after deactivation")
	}
	if u.DeactivatedAt() == nil {
		t.Error("deactivatedAt should be set after deactivation")
	}
}

func TestUser_Deactivate_EmailBecomesZeroValue(t *testing.T) {
	u := makeTestUser(t)
	if err := u.Deactivate("anom789", testNow); err != nil {
		t.Fatalf("Deactivate failed: %v", err)
	}
	if u.Email().String() != "" {
		t.Errorf("email after deactivation should be empty, got %q", u.Email().String())
	}
}

func TestUser_Deactivate_FailsIfAlreadyDeactivated(t *testing.T) {
	u := makeTestUser(t)
	if err := u.Deactivate("first", testNow); err != nil {
		t.Fatalf("first Deactivate failed: %v", err)
	}
	err := u.Deactivate("second", testNow)
	var appErr *apperror.AppError
	if !errors.As(err, &appErr) || appErr.Code != user.ErrCodeAlreadyDeactivated {
		t.Errorf("expected ALREADY_DEACTIVATED, got %v", err)
	}
}

func TestRestoreUser_ActiveUser(t *testing.T) {
	emailStr := "active@example.com"
	u := user.RestoreUser(
		"restore-id", &emailStr, "hash", "Jane", "jane-test",
		"CO", "Bogotá", "UFPS", "CONTESTANT", "ACTIVE",
		testNow, nil, nil,
	)
	if u.Email().String() != "active@example.com" {
		t.Errorf("Email = %q, want active@example.com", u.Email().String())
	}
	if u.Status() != user.StatusActive {
		t.Errorf("Status = %v, want StatusActive", u.Status())
	}
}

func TestRestoreUser_DeactivatedUserHasNoEmail(t *testing.T) {
	u := user.RestoreUser(
		"deact-id", nil, "hash", "Anonymous", "user_anonimo_xyz",
		"CO", "Bogotá", "UFPS", "CONTESTANT", "DEACTIVATED",
		testNow, nil, nil,
	)
	if u.Email().String() != "" {
		t.Errorf("deactivated user email should be empty, got %q", u.Email().String())
	}
	if u.Status() != user.StatusDeactivated {
		t.Errorf("Status = %v, want StatusDeactivated", u.Status())
	}
}
