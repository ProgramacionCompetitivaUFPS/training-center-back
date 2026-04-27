package group

import (
	"context"

	"github.com/training-judge-center/backend/pkg/apperror"
)

type fakeNicknameResolver struct {
	users map[string]*UserInfo
}

func (f *fakeNicknameResolver) ResolveByNickname(_ context.Context, nickname string) (*UserInfo, error) {
	if u, ok := f.users[nickname]; ok {
		return u, nil
	}
	return nil, nil
}

type fakeTxManager struct{}

func (f *fakeTxManager) WithTx(ctx context.Context, fn func(context.Context) error) error {
	return fn(ctx)
}

func assertErrCode(t interface {
	Helper()
	Fatalf(string, ...any)
}, err error, code string) {
	t.Helper()
	ae, ok := err.(*apperror.AppError)
	if !ok || ae.Code != code {
		t.Fatalf("expected error code %q, got %v", code, err)
	}
}
