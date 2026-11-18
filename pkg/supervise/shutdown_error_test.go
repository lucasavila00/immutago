package supervise

import (
	"context"
	"errors"
	"testing"
	"testing/synctest"
)

// stalled is a service whose Run ignores its context until the test releases it.
type stalled struct {
	Service[chan struct{}]
}

// Run waits for the release channel.
func (p *stalled) Run(context.Context) { <-p.Args() }

// expectShutdownWaiting checks that Shutdown times out and names the given wait.
func expectShutdownWaiting(t *testing.T, s *Supervisor, want string) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()
	err := s.Shutdown(ctx)
	var shutdown *ShutdownError
	if !errors.As(err, &shutdown) || !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("Shutdown = %v; want a ShutdownError that wraps %v", err, context.DeadlineExceeded)
	}
	if shutdown.Waiting != want {
		t.Errorf("Waiting = %q; want %q", shutdown.Waiting, want)
	}
}

// TestShutdownErrorNamesService names the service whose Run did not return, at each depth.
func TestShutdownErrorNamesService(t *testing.T) {
	cases := map[string]func(t *testing.T, top *Supervisor) *Supervisor{
		"root":   func(_ *testing.T, top *Supervisor) *Supervisor { return top },
		"nested": func(_ *testing.T, top *Supervisor) *Supervisor { return StartSupervisor(top) },
		"owned": func(t *testing.T, top *Supervisor) *Supervisor {
			owner, err := Start[initializing](top, &initHooks{init: func(*initializing) error { return nil }})
			if err != nil {
				t.Fatal(err)
			}
			return owner.Borrow().supervisor
		},
	}
	for name, parent := range cases {
		t.Run(name, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				top := NewRoot(nil)
				startRun(t, top, func(ctx context.Context) { <-ctx.Done() })
				release := make(chan struct{})
				if _, err := Start[stalled](parent(t, top), release); err != nil {
					t.Fatal(err)
				}
				synctest.Wait()
				expectShutdownWaiting(t, top, "supervise.stalled")
				close(release)
				expectShutdown(t, top)
			})
		})
	}
}

// TestShutdownErrorNamesLaterService names the next service after the first one stops.
func TestShutdownErrorNamesLaterService(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		top := NewRoot(nil)
		release, last := make(chan struct{}), make(chan struct{})
		startRun(t, top, func(context.Context) { <-release })
		if _, err := Start[stalled](top, last); err != nil {
			t.Fatal(err)
		}
		synctest.Wait()
		expectShutdownWaiting(t, top, "supervise.stalled")
		close(last)
		synctest.Wait()
		expectShutdownWaiting(t, top, "supervise.initializing")
		close(release)
		expectShutdown(t, top)
	})
}

// TestShutdownErrorNamesInit reports an accepted Init that did not return.
func TestShutdownErrorNamesInit(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		top := NewRoot(nil)
		entered, release := make(chan struct{}), make(chan struct{})
		started := make(chan error, 1)
		go func() {
			_, err := Start[initializing](top, &initHooks{
				init: func(*initializing) error { close(entered); <-release; return nil },
			})
			started <- err
		}()
		<-entered
		expectShutdownWaiting(t, top, "accepted Init calls")
		close(release)
		if err := <-started; err != nil {
			t.Error(err)
		}
		expectShutdown(t, top)
	})
}

// TestShutdownErrorText keeps the service name and the context error in the message.
func TestShutdownErrorText(t *testing.T) {
	named := &ShutdownError{Waiting: "pkg.Worker", Err: context.DeadlineExceeded}
	if got, want := named.Error(), "supervise: shutdown waited for pkg.Worker: context deadline exceeded"; got != want {
		t.Errorf("Error = %q; want %q", got, want)
	}
	unnamed := &ShutdownError{Err: context.Canceled}
	if got, want := unnamed.Error(), "supervise: shutdown did not complete: context canceled"; got != want {
		t.Errorf("Error = %q; want %q", got, want)
	}
}

// TestNewRootNoPanic stops the tree after a failure and returns the failure from Shutdown.
func TestNewRootNoPanic(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		reports := make(chan any, 2)
		top := NewRootNoPanic(func(value any, _ []byte) { reports <- value })
		sibling := make(chan struct{})
		startRun(t, top, func(ctx context.Context) { <-ctx.Done(); close(sibling) })
		startRun(t, top, func(context.Context) { panic("first failure") })
		<-top.stopped
		<-sibling
		if failure, _ := (<-reports).(*Failure); failure == nil || failure.Value != "first failure" {
			t.Errorf("report = %v; want the first Failure", failure)
		}
		for range 2 {
			var failure *Failure
			if err := top.Shutdown(context.Background()); !errors.As(err, &failure) || failure.Value != "first failure" {
				t.Errorf("Shutdown = %v; want the first Failure", err)
			}
		}
		_, err := Start[initializing](top, &initHooks{init: func(*initializing) error { return nil }})
		if !errors.Is(err, context.Canceled) {
			t.Errorf("Start after the failure = %v; want %v", err, context.Canceled)
		}
		if len(reports) != 0 {
			t.Errorf("second report %v", <-reports)
		}
	})
}

// TestNewRootNoPanicCleanStop returns nil from Shutdown when no service failed.
func TestNewRootNoPanicCleanStop(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		top := NewRootNoPanic(nil)
		startRun(t, top, func(ctx context.Context) { <-ctx.Done() })
		synctest.Wait()
		expectShutdown(t, top)
		nested := StartSupervisor(top)
		expectShutdown(t, nested)
	})
}

// TestNewRootNoPanicTimeout returns a ShutdownError while cleanup after a failure continues.
func TestNewRootNoPanicTimeout(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		top := NewRootNoPanic(func(any, []byte) {})
		release := make(chan struct{})
		if _, err := Start[stalled](top, release); err != nil {
			t.Fatal(err)
		}
		startRun(t, top, func(context.Context) { panic(context.Canceled) })
		synctest.Wait()
		expectShutdownWaiting(t, top, "supervise.stalled")
		close(release)
		err := top.Shutdown(context.Background())
		var failure *Failure
		var shutdown *ShutdownError
		if !errors.As(err, &failure) || errors.As(err, &shutdown) {
			t.Errorf("Shutdown = %v; want the Failure and no ShutdownError", err)
		}
		if !errors.Is(err, context.Canceled) {
			t.Errorf("Shutdown = %v; want the wrapped panic value", err)
		}
	})
}

// TestNewRootNoPanicNested returns nil from a nested Shutdown after the root failed.
func TestNewRootNoPanicNested(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		top := NewRootNoPanic(func(any, []byte) {})
		nested := StartSupervisor(top)
		startRun(t, nested, func(ctx context.Context) { <-ctx.Done() })
		startRun(t, top, func(context.Context) { panic("first failure") })
		<-top.stopped
		expectShutdown(t, nested)
	})
}

// TestShutdownErrorEndedContext starts shutdown and returns a ShutdownError when ctx has already ended.
func TestShutdownErrorEndedContext(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		top := NewRoot(nil)
		release := make(chan struct{})
		if _, err := Start[stalled](top, release); err != nil {
			t.Fatal(err)
		}
		synctest.Wait()
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		var shutdown *ShutdownError
		if err := top.Shutdown(ctx); !errors.As(err, &shutdown) || !errors.Is(err, context.Canceled) {
			t.Errorf("Shutdown = %v; want a ShutdownError that wraps %v", err, context.Canceled)
		}
		close(release)
		expectShutdown(t, top)
	})
}

// TestShutdownErrorConcurrent reads the wait from several callers while children stop in turn.
func TestShutdownErrorConcurrent(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		top := NewRoot(nil)
		releases := make([]chan struct{}, 4)
		for i := range releases {
			releases[i] = make(chan struct{})
			if _, err := Start[stalled](StartSupervisor(top), releases[i]); err != nil {
				t.Fatal(err)
			}
		}
		synctest.Wait()
		for i := len(releases) - 1; i >= 0; i-- {
			results := make(chan error, 4)
			for range 4 {
				go func() {
					ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
					defer cancel()
					results <- top.Shutdown(ctx)
				}()
			}
			for range 4 {
				var shutdown *ShutdownError
				if err := <-results; !errors.As(err, &shutdown) || shutdown.Waiting != "supervise.stalled" {
					t.Errorf("Shutdown = %v; want a wait for supervise.stalled", err)
				}
			}
			close(releases[i])
			synctest.Wait()
		}
		expectShutdown(t, top)
	})
}
