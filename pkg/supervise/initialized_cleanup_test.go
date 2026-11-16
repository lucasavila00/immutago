package supervise

import (
	"context"
	"errors"
	"runtime"
	"slices"
	"sync/atomic"
	"testing"
	"testing/synctest"
)

// TestInitializedServiceCleanup gives each successful Init exactly one Run for cleanup.
func TestInitializedServiceCleanup(t *testing.T) {
	for _, state := range []string{"admitted", "pending", "terminal"} {
		t.Run(state, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) { checkInitializedServiceCleanup(t, state) })
		})
	}
}

// checkInitializedServiceCleanup checks one setup resource through group shutdown.
func checkInitializedServiceCleanup(t *testing.T, state string) {
	t.Helper()
	top := root(func(any) {}, func(any, []byte) {})
	parent := top
	if state == "pending" {
		parent = newGroup(top, false)
	}
	var open, runs atomic.Int32
	_, err := Start[initializing](parent, &initHooks{
		init: func(*initializing) error { open.Add(1); return nil },
		run: func(_ *initializing, ctx context.Context) {
			defer open.Add(-1)
			runs.Add(1)
			if state == "pending" && ctx.Err() == nil {
				t.Error("cleanup Run started with an active context")
			}
			<-ctx.Done()
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if state == "terminal" {
		top.fail(&Failure{Service: "peer", Value: "failed"})
	}
	<-parent.Stop()
	<-top.Stop()
	if open.Load() != 0 || runs.Load() != 1 {
		t.Fatalf("open resources = %d, Run calls = %d; want 0 and 1", open.Load(), runs.Load())
	}
}

// TestPendingCleanupOrder ends staged children before the parent's cancelled Run.
func TestPendingCleanupOrder(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		top := NewRoot(nil)
		parent := newGroup(top, false)
		var order []int
		child := func(number int) *initHooks {
			return &initHooks{
				init: func(*initializing) error { return nil },
				run: func(_ *initializing, ctx context.Context) {
					if ctx.Err() == nil {
						t.Error("staged child activated during shutdown")
					}
					order = append(order, number)
				},
			}
		}
		_, err := Start[initializing](parent, &initHooks{
			init: func(p *initializing) error {
				for _, number := range []int{1, 2} {
					if _, err := Start[initializing](p.UncheckedChildren(), child(number)); err != nil {
						return err
					}
				}
				return nil
			},
			run: child(0).run,
		})
		if err != nil {
			t.Fatal(err)
		}
		<-parent.Stop()
		<-top.Stop()
		if !slices.Equal(order, []int{2, 1, 0}) {
			t.Fatalf("cleanup order = %v; want [2 1 0]", order)
		}
	})
}

// TestPendingCleanupFailure keeps a cleanup panic or Goexit terminal.
func TestPendingCleanupFailure(t *testing.T) {
	for _, mode := range []string{"panic", "Goexit"} {
		t.Run(mode, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) { checkPendingCleanupFailure(t, mode) })
		})
	}
}

// checkPendingCleanupFailure checks cleanup and one report for a pending service.
func checkPendingCleanupFailure(t *testing.T, mode string) {
	t.Helper()
	var reports atomic.Int32
	terminal := make(chan any, 1)
	top := root(func(value any) { terminal <- value },
		func(any, []byte) { reports.Add(1) })
	var open atomic.Int32
	child := &initHooks{
		init: func(*initializing) error { open.Add(1); return nil },
		run: func(_ *initializing, ctx context.Context) {
			defer open.Add(-1)
			if ctx.Err() == nil {
				t.Error("pending cleanup activated its Run")
			}
			if mode == "Goexit" {
				runtime.Goexit()
			}
			panic("cleanup")
		},
	}
	setupError := errors.New("parent setup failed")
	_, err := Start[initializing](top, &initHooks{
		init: func(p *initializing) error {
			if _, err := Start[initializing](p.UncheckedChildren(), child); err != nil {
				return err
			}
			return setupError
		},
	})
	if err != setupError {
		t.Fatalf("parent Init error = %v", err)
	}
	<-top.Stop()
	if failure := (<-terminal).(*Failure); failure.Value == nil {
		t.Fatal("cleanup lost its failure")
	}
	if open.Load() != 0 || reports.Load() != 1 {
		t.Fatalf("open resources = %d, reports = %d; want 0 and 1", open.Load(), reports.Load())
	}
}
