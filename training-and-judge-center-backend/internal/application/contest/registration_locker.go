package contest

import "context"

// RegistrationLocker serializes the registration changes of one contest. It must be
// called first inside a transaction: the lock lasts until that transaction ends.
type RegistrationLocker interface {
	Lock(ctx context.Context, contestID string) error
}
