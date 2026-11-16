package supervise

import (
	"context"
	"errors"
	"runtime"
	"testing"
	"testing/synctest"
	"time"
)

// TestFirstRunAfterPolicyCancellation checks an entry launched before policy cancellation.
func TestFirstRunAfterPolicyCancellation(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		top := NewRoot(nil)
		parent := newGroup(top, false)
		runs := 0
		_, err := Start[initializing](parent, &initHooks{
			init: func(*initializing) error { return nil },
			run: func(_ *initializing, ctx context.Context) {
				runs++
				if ctx.Err() == nil {
					t.Error("first attempt entered Run after its policy stopped with a live context")
				}
			},
		})
		if err != nil {
			t.Fatal(err)
		}
		entry := parent.children.first
		run := entry.run
		parent.forget(entry)
		parent.cancel()
		run(context.Background())
		<-parent.Stop()
		<-top.Stop()
		if runs != 1 {
			t.Fatalf("first attempt ran %d times; want 1", runs)
		}
	})
}

// TestCleanupTimeDoesNotExtendRetryWindow measures failures before a long cleanup wait.
func TestCleanupTimeDoesNotExtendRetryWindow(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		parent := NewRoot(nil)
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		done := make(chan struct{})
		runs := 0
		run := func(ctx context.Context) {
			runs++
			if runs <= 2 {
				panic("retry")
			}
			<-ctx.Done()
		}
		cleanup := func() {
			if runs == 1 {
				time.Sleep(restartWindow)
			}
		}
		go func() {
			defer close(done)
			watch(parent, ctx, "worker", run, cleanup)
		}()
		time.Sleep(restartWindow + 2*time.Second)
		synctest.Wait()
		if runs != 3 {
			t.Errorf("Run count = %d; want 3 after two one-second delays", runs)
		}
		cancel()
		<-done
		<-parent.Stop()
	})
}

// TestTerminalBeforeOwnChildCleanup covers failure observation inside the failing service.
func TestTerminalBeforeOwnChildCleanup(t *testing.T) {
	for _, mode := range []string{"Run", "Init", "Goexit"} {
		t.Run(mode, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) { checkTerminalBeforeOwnChildCleanup(t, mode) })
		})
	}
}

// checkTerminalBeforeOwnChildCleanup holds one terminal attempt's child open.
func checkTerminalBeforeOwnChildCleanup(t *testing.T, mode string) {
	t.Helper()
	reported, release := make(chan struct{}), make(chan struct{})
	top := root(func(any) {}, func(value any, _ []byte) {
		if _, terminal := value.(*Failure); terminal {
			close(reported)
		}
	})
	inits := 0
	_, err := Start[initializing](top, &initHooks{
		init: func(p *initializing) error {
			inits++
			terminal := mode == "Goexit" || inits == restartLimit+1
			_, err := Start[initializing](p.Children(), &initHooks{
				init: func(*initializing) error { return nil },
				run: func(_ *initializing, ctx context.Context) {
					<-ctx.Done()
					if terminal {
						<-release
					}
				},
			})
			if err != nil {
				return err
			}
			if mode == "Init" && inits > 1 {
				return errors.New("restart setup failed")
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
	time.Sleep(31 * time.Second)
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
