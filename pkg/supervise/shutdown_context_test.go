package supervise

import (
	"context"
	"errors"
	"testing"
	"testing/synctest"
	"time"
)

const testTimeout = time.Second

// startRun starts one service with the given Run function and no setup.
func startRun(t *testing.T, parent *Supervisor, run func(context.Context)) {
	t.Helper()
	_, err := Start[initializing](parent, &initHooks{
		init: func(*initializing) error { return nil },
		run:  func(_ *initializing, ctx context.Context) { run(ctx) },
	})
	if err != nil {
		t.Error(err)
	}
}

// expectShutdownTimeout checks that Shutdown returns context.DeadlineExceeded when its timeout ends.
func expectShutdownTimeout(t *testing.T, s *Supervisor) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()
	at := time.Now()
	if err := s.Shutdown(ctx); !errors.Is(err, context.DeadlineExceeded) {
		t.Errorf("Shutdown = %v; want %v", err, context.DeadlineExceeded)
	}
	if elapsed := time.Since(at); elapsed != testTimeout {
		t.Errorf("Shutdown returned after %s; want %s", elapsed, testTimeout)
	}
}

// expectShutdown checks that Shutdown returns nil without a time limit.
func expectShutdown(t *testing.T, s *Supervisor) {
	t.Helper()
	if err := s.Shutdown(context.Background()); err != nil {
		t.Errorf("Shutdown = %v; want nil", err)
	}
}

// TestShutdownInTime returns nil and keeps the reverse stop order.
func TestShutdownInTime(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		top := NewRoot(nil)
		order := make(chan int, 3)
		for i := range 3 {
			startRun(t, top, func(ctx context.Context) { <-ctx.Done(); order <- i })
		}
		synctest.Wait()
		ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
		defer cancel()
		at := time.Now()
		if err := top.Shutdown(ctx); err != nil {
			t.Errorf("Shutdown = %v; want nil", err)
		}
		if elapsed := time.Since(at); elapsed != 0 {
			t.Errorf("Shutdown returned after %s; want 0s", elapsed)
		}
		for want := 2; want >= 0; want-- {
			if got := <-order; got != want {
				t.Errorf("service %d stopped; want %d", got, want)
			}
		}
	})
}

// TestShutdownIgnoredCancellation returns at the deadline and lets shutdown continue.
func TestShutdownIgnoredCancellation(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		top := NewRoot(nil)
		release := make(chan struct{})
		startRun(t, top, func(context.Context) { <-release })
		expectShutdownTimeout(t, top)
		close(release)
		expectShutdown(t, top)
	})
}

// TestShutdownCancelledContext returns context.Canceled when its context is cancelled.
func TestShutdownCancelledContext(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		top := NewRoot(nil)
		release := make(chan struct{})
		startRun(t, top, func(context.Context) { <-release })
		ctx, cancel := context.WithCancel(context.Background())
		result := make(chan error, 1)
		go func() { result <- top.Shutdown(ctx) }()
		synctest.Wait()
		if len(result) != 0 {
			t.Errorf("Shutdown = %v before cancellation", <-result)
		}
		cancel()
		if err := <-result; !errors.Is(err, context.Canceled) {
			t.Errorf("Shutdown = %v; want %v", err, context.Canceled)
		}
		close(release)
		expectShutdown(t, top)
	})
}

// TestShutdownStoppedEndedContext prefers completion when both channels are ready.
func TestShutdownStoppedEndedContext(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		top := NewRoot(nil)
		startRun(t, top, func(ctx context.Context) { <-ctx.Done() })
		<-top.Stop()
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		if err := top.Shutdown(ctx); err != nil {
			t.Errorf("Shutdown = %v; want nil", err)
		}
	})
}

// TestShutdownEndedContextStarts starts shutdown before it returns an ended context's error.
func TestShutdownEndedContextStarts(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		top := NewRoot(nil)
		cancelled, release := make(chan struct{}), make(chan struct{})
		startRun(t, top, func(ctx context.Context) { <-ctx.Done(); close(cancelled); <-release })
		synctest.Wait()
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		if err := top.Shutdown(ctx); !errors.Is(err, context.Canceled) {
			t.Errorf("Shutdown = %v; want %v", err, context.Canceled)
		}
		<-cancelled
		close(release)
		expectShutdown(t, top)
	})
}

// TestShutdownConcurrentCalls waits for one shutdown from concurrent Shutdown and Stop calls.
func TestShutdownConcurrentCalls(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		top := NewRoot(nil)
		release := make(chan struct{})
		runs := make(chan struct{}, 2)
		startRun(t, top, func(ctx context.Context) { <-ctx.Done(); <-release; runs <- struct{}{} })
		synctest.Wait()
		results := make(chan error, 3)
		stops := make(chan (<-chan struct{}), 2)
		for range 3 {
			go func() { results <- top.Shutdown(context.Background()) }()
		}
		for range 2 {
			go func() { stops <- top.Stop() }()
		}
		synctest.Wait()
		if len(results) != 0 {
			t.Errorf("Shutdown = %v before shutdown completed", <-results)
		}
		close(release)
		for range 3 {
			if err := <-results; err != nil {
				t.Errorf("Shutdown = %v; want nil", err)
			}
		}
		for range 2 {
			if done := <-stops; done != top.stopped {
				t.Error("Stop returned another channel")
			}
		}
		<-top.stopped
		if len(runs) != 1 {
			t.Errorf("Run ended %d times; want 1", len(runs))
		}
	})
}

// TestShutdownNested stops only the subtree of a nested or service-owned supervisor.
func TestShutdownNested(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		top := NewRoot(nil)
		sibling := make(chan struct{})
		startRun(t, top, func(ctx context.Context) { <-ctx.Done(); close(sibling) })
		nested := StartSupervisor(top)
		nestedStopped := make(chan struct{})
		startRun(t, nested, func(ctx context.Context) { <-ctx.Done(); close(nestedStopped) })
		owner, err := Start[initializing](top, &initHooks{init: func(*initializing) error { return nil }})
		if err != nil {
			t.Fatal(err)
		}
		owned := owner.Borrow().supervisor
		ownedStopped := make(chan struct{})
		startRun(t, owned, func(ctx context.Context) { <-ctx.Done(); close(ownedStopped) })
		synctest.Wait()
		expectShutdown(t, nested)
		<-nestedStopped
		expectShutdown(t, owned)
		<-ownedStopped
		synctest.Wait()
		select {
		case <-sibling:
			t.Error("a nested Shutdown stopped a sibling service")
		default:
		}
		if err := top.admission.Err(); err != nil {
			t.Errorf("root admission = %v; want open", err)
		}
		expectShutdown(t, top)
		<-sibling
	})
}

// TestShutdownRootFailureWaits returns at the deadline while a Run ignores its context after a failure.
func TestShutdownRootFailureWaits(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		exited, release := make(chan any, 2), make(chan struct{})
		top := root(func(value any) { exited <- value }, func(any, []byte) {})
		startRun(t, top, func(context.Context) { <-release })
		startRun(t, top, func(context.Context) { panic("first failure") })
		synctest.Wait()
		expectShutdownTimeout(t, top)
		if len(exited) != 0 {
			t.Errorf("terminal action %v before shutdown completed", <-exited)
		}
		close(release)
		if failure, _ := (<-exited).(*Failure); failure == nil || failure.Value != "first failure" {
			t.Errorf("terminal value = %v; want the first Failure", failure)
		}
		<-top.stopped
		if len(exited) != 0 {
			t.Errorf("second terminal action %v", <-exited)
		}
	})
}

// TestShutdownRootFailurePrompt calls the terminal action once when shutdown completes.
// The test exit hook returns, so the root then closes its channel and Shutdown returns the failure.
// The panic in NewRoot's exit ends the process before the channel closes.
func TestShutdownRootFailurePrompt(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		exited := make(chan any, 2)
		top := root(func(value any) { exited <- value }, func(any, []byte) {})
		startRun(t, top, func(context.Context) { panic("first failure") })
		if failure, _ := (<-exited).(*Failure); failure == nil || failure.Value != "first failure" {
			t.Errorf("terminal value = %v; want the first Failure", failure)
		}
		ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
		defer cancel()
		var failure *Failure
		if err := top.Shutdown(ctx); !errors.As(err, &failure) || failure.Value != "first failure" {
			t.Errorf("Shutdown = %v; want the first Failure with the test exit hook", err)
		}
		if len(exited) != 0 {
			t.Errorf("second terminal action %v", <-exited)
		}
	})
}

// TestShutdownBlockedInit returns at the deadline while an accepted Init blocks.
func TestShutdownBlockedInit(t *testing.T) {
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
		expectShutdownTimeout(t, top)
		close(release)
		if err := <-started; err != nil {
			t.Error(err)
		}
		expectShutdown(t, top)
	})
}
