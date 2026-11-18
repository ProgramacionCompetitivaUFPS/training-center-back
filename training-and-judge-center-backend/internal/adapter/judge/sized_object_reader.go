package judge

import "context"

// sizedObjectReader is a storage reader that can also say how big an object is
// before it is read, which is what lets the test data cache reserve disk for it.
type sizedObjectReader interface {
	gcsReader
	objectSize(ctx context.Context, object string) (int64, error)
}
