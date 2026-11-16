package supervise

import (
	"context"
	"runtime"
	"sync/atomic"
	"testing"
	"testing/synctest"
)

// TestInitializedAttemptCleanup gives each successful Init exactly one Run for cleanup.
func TestInitializedAttemptCleanup(t *testing.T) {
	for _, state := range []string{"stopped", "admitted", "pending", "terminal"} {
		t.Run(state, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				top := root(func(any) {}, func(any, []byte) {})
				parent := top
				if state == "pending" {
					parent = newGroup(top, false)
				}
				if state == "stopped" {
					<-parent.Stop()
				}
				var open, runs atomic.Int32
				_, err := Start[initializing](parent, &initHooks{
					init: func(*initializing) error { open.Add(1); return nil },
					run: func(_ *initializing, ctx context.Context) {
						defer open.Add(-1)
						runs.Add(1)
						if (state == "stopped" || state == "pending") && ctx.Err() == nil {
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
			})
		})
	}
}

// TestPendingCleanupFailure keeps a cleanup panic or Goexit inside its watcher.
func TestPendingCleanupFailure(t *testing.T) {
	for _, mode := range []string{"panic", "Goexit"} {
		t.Run(mode, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				var reports atomic.Int32
				top := root(func(any) { t.Error("stopped attempt became terminal") },
					func(any, []byte) { reports.Add(1) })
				parent := newGroup(top, false)
				var open atomic.Int32
				_, err := Start[initializing](parent, &initHooks{
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
				})
				if err != nil {
					t.Fatal(err)
				}
				<-parent.Stop()
				<-top.Stop()
				if open.Load() != 0 || reports.Load() != 1 {
					t.Fatalf("open resources = %d, reports = %d; want 0 and 1", open.Load(), reports.Load())
				}
			})
		})
	}
}
