package supervise

import (
	"context"
	"runtime"
	"testing"
	"testing/synctest"
)

// TestRunAfterAdmissionCloses keeps a delayed Run in shutdown order.
func TestRunAfterAdmissionCloses(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		top := NewRoot(nil)
		parent := newSupervisor(top, false)
		inits, runs := 0, 0
		ref, err := Start[initializing](parent, &initHooks{
			init: func(*initializing) error { inits++; return nil },
			run: func(_ *initializing, ctx context.Context) {
				runs++
				if ctx.Err() == nil {
					t.Error("service entered Run after admission closed with a live context")
				}
			},
		})
		if err != nil {
			t.Fatal(err)
		}
		original := ref.Borrow()
		entry := parent.children.first
		run := entry.run
		entered, release := make(chan struct{}), make(chan struct{})
		entry.run = func(ctx context.Context) { close(entered); <-release; run(ctx) }
		parent.mu.Lock()
		parent.activate()
		parent.mu.Unlock()
		<-entered
		cleanup, slowRelease := make(chan struct{}), make(chan struct{})
		_, err = Start[initializing](parent, &initHooks{
			init: func(*initializing) error { return nil },
			run: func(_ *initializing, ctx context.Context) {
				<-ctx.Done()
				close(cleanup)
				<-slowRelease
			},
		})
		if err != nil {
			t.Fatal(err)
		}
		synctest.Wait()
		done := parent.Stop()
		<-cleanup
		close(release)
		synctest.Wait()
		if runs != 0 {
			t.Error("service ran cleanup before the newer service stopped")
		}
		close(slowRelease)
		<-done
		<-top.Stop()
		if inits != 1 || runs != 1 || ref.Borrow() != original {
			t.Fatalf("service: Init calls=%d, Run calls=%d, changed=%t", inits, runs, ref.Borrow() != original)
		}
	})
}

// TestTerminalBeforeOwnChildCleanup covers failure observation inside the failing service.
func TestTerminalBeforeOwnChildCleanup(t *testing.T) {
	for _, mode := range []string{"Run", "Goexit"} {
		t.Run(mode, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) { checkTerminalBeforeOwnChildCleanup(t, mode) })
		})
	}
}

// checkTerminalBeforeOwnChildCleanup holds one failed service's child open.
func checkTerminalBeforeOwnChildCleanup(t *testing.T, mode string) {
	t.Helper()
	reported, release := make(chan struct{}), make(chan struct{})
	top := root(func(any) {}, func(value any, _ []byte) {
		if _, terminal := value.(*Failure); terminal {
			close(reported)
		}
	})
	_, err := Start[initializing](top, &initHooks{
		init: func(p *initializing) error {
			_, err := Start[initializing](p.supervisor, &initHooks{
				init: func(*initializing) error { return nil },
				run: func(_ *initializing, ctx context.Context) {
					<-ctx.Done()
					<-release
				},
			})
			if err != nil {
				return err
			}
			return nil
		},
		run: func(*initializing, context.Context) {
			if mode == "Goexit" {
				runtime.Goexit()
			}
			panic("Run failed")
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	synctest.Wait()
	select {
	case <-reported:
	default:
		t.Error("terminal failure was hidden by its own child cleanup")
	}
	close(release)
	synctest.Wait()
	<-top.Stop()
}
