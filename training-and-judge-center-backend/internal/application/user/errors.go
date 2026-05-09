package user

import "github.com/training-judge-center/backend/pkg/apperror"

var errInvalidRecoveryAttempt = apperror.NewBadRequest("INVALID_RECOVERY_ATTEMPT", "Invalid email or recovery code")
