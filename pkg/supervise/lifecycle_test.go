package supervise

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"testing/synctest"
	"time"
)

// TestReturnedRunKeepsChildFailure checks a failure observed before normal cleanup.
func TestReturnedRunKeepsChildFailure(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		group := NewRoot(nil)
		defer func() { <-group.Stop() }()
		cleanup, release, finish := make(chan struct{}), make(chan struct{}), make(chan struct{})
		var runs atomic.Int32
		hooks := &initHooks{
			init: func(*initializing) error { return nil },
			run: func(p *initializing, ctx context.Context) {
				if runs.Add(1) != 1 {
					<-ctx.Done()
					return
				}
				_, _ = Start[initializing](p.Children(), AssumeConcurrentSafe(&initHooks{
					init: func(*initializing) error { return nil },
					run:  func(*initializing, context.Context) { panic("child failure") },
				}))
				_, _ = Start[initializing](p.Children(), AssumeConcurrentSafe(&initHooks{
					init: func(*initializing) error { return nil },
					run: func(_ *initializing, ctx context.Context) {
						<-ctx.Done()
						close(cleanup)
						<-release
					},
				}))
				<-finish
			},
		}
		_, _ = Start[initializing](group, AssumeConcurrentSafe(hooks))
		<-cleanup
		close(finish)
		synctest.Wait()
		close(release)
		synctest.Wait()
		time.Sleep(time.Second)
		synctest.Wait()
		if got := runs.Load(); got != 2 {
			t.Fatalf("parent runs = %d, want 2 after child failure", got)
		}
	})
}

// TestStopDuringFailureCleanup checks stop after a failure was selected.
func TestStopDuringFailureCleanup(t *testing.T) {
	for _, nested := range []bool{false, true} {
		t.Run(map[bool]string{false: "root", true: "nested"}[nested], func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				var escalations atomic.Int32
				parent := root(func(any) { escalations.Add(1) }, nil)
				group := parent
				if nested {
					group = StartGroup(parent)
				}
				cleanup, release := make(chan struct{}), make(chan struct{})
				_, _ = Start[initializing](group, AssumeConcurrentSafe(&initHooks{
					init: func(*initializing) error { return nil },
					run:  func(*initializing, context.Context) { panic("child failure") },
				}))
				_, _ = Start[initializing](group, AssumeConcurrentSafe(&initHooks{
					init: func(*initializing) error { return nil },
					run: func(_ *initializing, ctx context.Context) {
						<-ctx.Done()
						close(cleanup)
						<-release
					},
				}))
				<-cleanup
				done := group.Stop()
				close(release)
				<-done
				time.Sleep(time.Minute)
				synctest.Wait()
				if got := escalations.Load(); got != 0 {
					t.Fatalf("root escalations after stop = %d", got)
				}
				<-parent.Stop()
			})
		})
	}
}

// TestAncestorStopEndsRestartPolicy checks propagation without early Run cancellation.
func TestAncestorStopEndsRestartPolicy(t *testing.T) {
	for _, where := range []string{"group", "Init", "Run"} {
		t.Run(where, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				parent := root(func(value any) { t.Errorf("root escalated: %v", value) }, nil)
				var inits atomic.Int32
				child := AssumeConcurrentSafe(&initHooks{
					init: func(*initializing) error { inits.Add(1); return nil },
					run:  func(*initializing, context.Context) { panic("child failure") },
				})
				ended := make(chan struct{})
				if where == "group" {
					nested := StartGroup(parent)
					_, _ = Start[initializing](nested, child)
					_, _ = Start[initializing](nested, AssumeConcurrentSafe(&initHooks{
						init: func(*initializing) error { return nil },
						run:  func(_ *initializing, ctx context.Context) { <-ctx.Done(); close(ended) },
					}))
				} else {
					_, _ = Start[initializing](parent, AssumeConcurrentSafe(&initHooks{
						init: func(p *initializing) error {
							if where == "Init" {
								_, _ = Start[initializing](p.Children(), child)
							}
							return nil
						},
						run: func(p *initializing, ctx context.Context) {
							if where == "Run" {
								_, _ = Start[initializing](p.Children(), child)
							}
							<-ctx.Done()
							close(ended)
						},
					}))
				}
				cleanup, release := make(chan struct{}), make(chan struct{})
				_, _ = Start[initializing](parent, AssumeConcurrentSafe(&initHooks{
					init: func(*initializing) error { return nil },
					run: func(_ *initializing, ctx context.Context) {
						<-ctx.Done()
						close(cleanup)
						<-release
					},
				}))
				time.Sleep(2 * time.Second)
				before := inits.Load()
				done := parent.Stop()
				<-cleanup
				time.Sleep(time.Minute)
				synctest.Wait()
				if got := inits.Load(); got != before {
					t.Errorf("Init count = %d, want %d", got, before)
				}
				select {
				case <-ended:
					t.Error("older Run context ended before newer child stopped")
				default:
				}
				close(release)
				<-done
			})
		})
	}
}

// TestStopDuringAdmittedRestartInit lets Init finish without publishing its candidate.
func TestStopDuringAdmittedRestartInit(t *testing.T) {
	for _, fail := range []bool{false, true} {
		t.Run(map[bool]string{false: "success", true: "error"}[fail], func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				parent := NewRoot(nil)
				entered := make(chan *initializing, 1)
				initRelease, slowRelease, cleanup := make(chan struct{}), make(chan struct{}), make(chan struct{})
				var inits, runs atomic.Int32
				ref, _ := Start[initializing](parent, AssumeConcurrentSafe(&initHooks{
					init: func(p *initializing) error {
						if inits.Add(1) > 1 {
							p.Children()
							entered <- p
							<-initRelease
							if fail {
								return errors.New("setup failure")
							}
						}
						return nil
					},
					run: func(*initializing, context.Context) { runs.Add(1); panic("retry") },
				}))
				original := ref.Borrow()
				candidate := <-entered
				_, _ = Start[initializing](parent, AssumeConcurrentSafe(&initHooks{
					init: func(*initializing) error { return nil },
					run: func(_ *initializing, ctx context.Context) {
						<-ctx.Done()
						close(cleanup)
						<-slowRelease
					},
				}))
				synctest.Wait()
				done := parent.Stop()
				<-cleanup
				close(initRelease)
				synctest.Wait()
				expectInitEnded(t, candidate)
				if ref.Borrow() != original || runs.Load() != 1 || inits.Load() != 2 {
					t.Errorf("restart after stop: changed=%t, runs=%d, Init calls=%d", ref.Borrow() != original, runs.Load(), inits.Load())
				}
				close(slowRelease)
				<-done
			})
		})
	}
}

// TestQueuedFailureAfterStop keeps the old child running until its shutdown turn.
func TestQueuedFailureAfterStop(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		group := root(func(value any) { t.Errorf("root escalated: %v", value) }, nil)
		ended, cleanup, release := make(chan struct{}), make(chan struct{}), make(chan struct{})
		_, _ = Start[initializing](group, AssumeConcurrentSafe(&initHooks{
			init: func(*initializing) error { return nil },
			run:  func(_ *initializing, ctx context.Context) { <-ctx.Done(); close(ended) },
		}))
		_, _ = Start[initializing](group, AssumeConcurrentSafe(&initHooks{
			init: func(*initializing) error { return nil },
			run:  func(_ *initializing, ctx context.Context) { <-ctx.Done(); close(cleanup); <-release },
		}))
		synctest.Wait()
		done := group.Stop()
		<-cleanup
		group.mu.Lock()
		failure := group.failure
		group.mu.Unlock()
		failure <- "queued failure"
		synctest.Wait()
		select {
		case <-ended:
			t.Error("queued failure bypassed stop order")
		default:
		}
		close(release)
		<-done
	})
}
