package supervise

import (
	"context"
	"testing"
	"testing/synctest"
	"time"
)

const testDeadline = time.Second

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

// checkDeadlineExit waits for the terminal action and checks its time and first failure.
func checkDeadlineExit(t *testing.T, exited <-chan any, at time.Time) {
	t.Helper()
	value := <-exited
	if elapsed := time.Since(at); elapsed != testDeadline {
		t.Errorf("terminal action after %s; want %s", elapsed, testDeadline)
	}
	if failure, _ := value.(*Failure); failure == nil || failure.Value != "first failure" {
		t.Errorf("terminal value = %v; want the first Failure", value)
	}
}

// checkNoExit waits past the deadline and checks that no terminal action follows.
func checkNoExit(t *testing.T, exited <-chan any) {
	t.Helper()
	time.Sleep(2 * testDeadline)
	if len(exited) != 0 {
		t.Errorf("unexpected terminal action %v", <-exited)
	}
}

// TestFailureDeadlineIgnoredCancellation ends a failed root while a Run ignores its context.
func TestFailureDeadlineIgnoredCancellation(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		exited, release := make(chan any, 2), make(chan struct{})
		top := root(func(value any) { exited <- value }, func(any, []byte) {}, FailureDeadline(testDeadline))
		startRun(t, top, func(context.Context) { <-release })
		at := time.Now()
		startRun(t, top, func(context.Context) { panic("first failure") })
		checkDeadlineExit(t, exited, at)
		close(release)
		<-top.Stop()
		checkNoExit(t, exited)
	})
}

// TestFailureDeadlineDisabled keeps the unbounded wait without a positive deadline.
func TestFailureDeadlineDisabled(t *testing.T) {
	cases := []struct {
		name    string
		options []RootOption
	}{
		{"none", nil},
		{"zero", []RootOption{FailureDeadline(0)}},
		{"negative", []RootOption{FailureDeadline(-testDeadline)}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				exited, release := make(chan any, 2), make(chan struct{})
				top := root(func(value any) { exited <- value }, func(any, []byte) {}, c.options...)
				startRun(t, top, func(context.Context) { <-release })
				startRun(t, top, func(context.Context) { panic("first failure") })
				time.Sleep(time.Hour)
				if len(exited) != 0 {
					t.Error("terminal action did not wait for Run")
				}
				close(release)
				<-top.Stop()
				if failure, _ := (<-exited).(*Failure); failure == nil || failure.Value != "first failure" {
					t.Errorf("terminal value = %v; want the first Failure", failure)
				}
				checkNoExit(t, exited)
			})
		})
	}
}

// TestFailureDeadlineBlockedReport ends a failed root while its reporter blocks.
func TestFailureDeadlineBlockedReport(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		exited, release := make(chan any, 2), make(chan struct{})
		top := root(func(value any) { exited <- value }, func(any, []byte) { <-release }, FailureDeadline(testDeadline))
		at := time.Now()
		startRun(t, top, func(context.Context) { panic("first failure") })
		checkDeadlineExit(t, exited, at)
		close(release)
		<-top.Stop()
		checkNoExit(t, exited)
	})
}

// TestFailureDeadlineBlockedInit ends a failed root while an accepted Init blocks.
func TestFailureDeadlineBlockedInit(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		exited, trigger := make(chan any, 2), make(chan struct{})
		top := root(func(value any) { exited <- value }, func(any, []byte) {}, FailureDeadline(testDeadline))
		startRun(t, top, func(context.Context) { <-trigger; panic("first failure") })
		entered, releaseInit := make(chan struct{}), make(chan struct{})
		started := make(chan error, 1)
		go func() {
			_, err := Start[initializing](top, &initHooks{
				init: func(*initializing) error { close(entered); <-releaseInit; return nil },
			})
			started <- err
		}()
		<-entered
		at := time.Now()
		close(trigger)
		checkDeadlineExit(t, exited, at)
		close(releaseInit)
		if err := <-started; err != nil {
			t.Error(err)
		}
		<-top.Stop()
		checkNoExit(t, exited)
	})
}

// TestFailureDeadlinePromptShutdown runs one terminal action when shutdown ends first.
func TestFailureDeadlinePromptShutdown(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		exited := make(chan any, 2)
		top := root(func(value any) { exited <- value }, func(any, []byte) {}, FailureDeadline(testDeadline))
		at := time.Now()
		startRun(t, top, func(context.Context) { panic("first failure") })
		value := <-exited
		if elapsed := time.Since(at); elapsed != 0 {
			t.Errorf("terminal action after %s; want 0s", elapsed)
		}
		if failure, _ := value.(*Failure); failure == nil || failure.Value != "first failure" {
			t.Errorf("terminal value = %v; want the first Failure", value)
		}
		<-top.Stop()
		checkNoExit(t, exited)
	})
}

// TestFailureDeadlineStop does not apply the deadline to Stop without a failure.
func TestFailureDeadlineStop(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		exited := make(chan any, 2)
		top := root(func(value any) { exited <- value }, nil, FailureDeadline(testDeadline))
		startRun(t, top, func(ctx context.Context) { <-ctx.Done() })
		synctest.Wait()
		<-top.Stop()
		checkNoExit(t, exited)
	})
}

// TestFailureDeadlineFirstFailure keeps the first failure when cleanup fails again.
func TestFailureDeadlineFirstFailure(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		exited, release := make(chan any, 2), make(chan struct{})
		top := root(func(value any) { exited <- value }, func(any, []byte) {}, FailureDeadline(testDeadline))
		startRun(t, top, func(context.Context) { <-release })
		startRun(t, top, func(ctx context.Context) { <-ctx.Done(); panic("second failure") })
		at := time.Now()
		startRun(t, top, func(context.Context) { panic("first failure") })
		checkDeadlineExit(t, exited, at)
		close(release)
		<-top.Stop()
		checkNoExit(t, exited)
	})
}
