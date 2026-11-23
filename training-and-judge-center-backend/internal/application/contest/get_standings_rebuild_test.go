package contest

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func newStandingsUseCaseCountingLoads(cache *mockStandingsCache, onLoad func(ctx context.Context)) *GetStandingsUseCase {
	return NewGetStandingsUseCase(
		repoWith(activeContest()),
		&mockRegistrationRepository{},
		&mockStandingsSubmissionProvider{listByContestFn: func(ctx context.Context, _ string) ([]ContestSubmissionData, error) {
			onLoad(ctx)
			return nil, nil
		}},
		&mockTeamParticipantProvider{},
		&mockParticipantProfileProvider{},
		&mockTeamDisplayProvider{},
		defaultProblemProvider(),
		&mockGroupProvider{findByIDFn: func(_ context.Context, _ string) (*GroupInfo, error) { return visibleGroup(), nil }},
		isMemberNotLead(),
		cache,
		testStaleness,
	)
}

func TestGetStandingsUseCase_ConcurrentCacheMissesShareOneRebuild(t *testing.T) {
	const callers = 50
	var loads atomic.Int32
	release := make(chan struct{})
	uc := newStandingsUseCaseCountingLoads(&mockStandingsCache{}, func(_ context.Context) {
		loads.Add(1)
		<-release
	})

	var started, done sync.WaitGroup
	started.Add(callers)
	done.Add(callers)
	errs := make(chan error, callers)
	for i := 0; i < callers; i++ {
		go func() {
			defer done.Done()
			started.Done()
			_, err := uc.Execute(context.Background(), GetStandingsInput{
				CurrentUser: asContestant(callerID),
				GroupID:     testGroupID,
				ContestID:   testContestID,
				Page:        1,
				Limit:       10,
			})
			errs <- err
		}()
	}
	started.Wait()
	time.Sleep(50 * time.Millisecond)
	close(release)
	done.Wait()
	close(errs)

	for err := range errs {
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	}
	if got := loads.Load(); got != 1 {
		t.Errorf("submissions were loaded %d times, want 1 shared rebuild", got)
	}
}

func TestGetStandingsUseCase_RebuildIsBoundedAndOutlivesTheCaller(t *testing.T) {
	var hadDeadline bool
	var callerGone bool
	uc := newStandingsUseCaseCountingLoads(&mockStandingsCache{}, func(ctx context.Context) {
		deadline, ok := ctx.Deadline()
		hadDeadline = ok && time.Until(deadline) <= standingsRebuildTimeout
		callerGone = ctx.Err() != nil
	})
	callerCtx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := uc.Execute(callerCtx, GetStandingsInput{
		CurrentUser: asContestant(callerID),
		GroupID:     testGroupID,
		ContestID:   testContestID,
		Page:        1,
		Limit:       10,
	})

	if err != nil {
		t.Fatalf("a cancelled caller must not fail the rebuild: %v", err)
	}
	if !hadDeadline {
		t.Errorf("the rebuild must run with a deadline of at most %s", standingsRebuildTimeout)
	}
	if callerGone {
		t.Error("the rebuild must not inherit the caller's cancellation")
	}
}

func TestStandingsRefreshLock_OutlastsTheRebuildTimeout(t *testing.T) {
	if standingsRefreshLockTTL <= standingsRebuildTimeout {
		t.Errorf("lock TTL %s must exceed the rebuild timeout %s, or a second holder can start mid-rebuild",
			standingsRefreshLockTTL, standingsRebuildTimeout)
	}
}

func TestGetStandingsUseCase_StaleRefreshHoldsTheLockForItsFullTTL(t *testing.T) {
	cached := &CachedStandings{LastUpdated: time.Now().Add(-time.Hour)}
	lockTTL := make(chan time.Duration, 1)
	cache := &mockStandingsCache{
		getFn: func(_ context.Context, _ string) (*CachedStandings, error) { return cached, nil },
		acquireRefreshLockFn: func(_ context.Context, _ string, ttl time.Duration) (bool, error) {
			lockTTL <- ttl
			return false, nil
		},
	}
	uc := newGetStandingsUseCase(activeContest(), nil, nil, cache, visibleGroup(), isMemberNotLead())

	_, err := uc.Execute(context.Background(), GetStandingsInput{
		CurrentUser: asContestant(callerID),
		GroupID:     testGroupID,
		ContestID:   testContestID,
		Page:        1,
		Limit:       10,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	select {
	case got := <-lockTTL:
		if got != standingsRefreshLockTTL {
			t.Errorf("lock TTL = %s, want %s (independent of the staleness threshold)", got, standingsRefreshLockTTL)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("expected a stale hit to try the refresh lock")
	}
}
