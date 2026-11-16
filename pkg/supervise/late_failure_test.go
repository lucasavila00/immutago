package supervise

import (
	"context"
	"runtime"
	"testing"
	"testing/synctest"
)

// TestLateRunFailure waits for cleanup and terminal action before late Start returns.
func TestLateRunFailure(t *testing.T) {
	for _, mode := range []string{"panic", "Goexit"} {
		for _, state := range []string{"stopping", "stopped"} {
			t.Run(mode+"/"+state, func(t *testing.T) {
				synctest.Test(t, func(t *testing.T) { checkLateRunFailure(t, mode, state) })
			})
		}
	}
}

// checkLateRunFailure holds Init, child cleanup, reporting, and exit in that order.
func checkLateRunFailure(t *testing.T, mode, state string) {
	t.Helper()
	initReady, releaseInit := make(chan struct{}), make(chan struct{})
	childCleanup, releaseChild := make(chan struct{}), make(chan struct{})
	reported, releaseReport := make(chan struct{}), make(chan struct{})
	exiting, releaseExit := make(chan *Failure, 1), make(chan struct{})
	started := make(chan error, 1)
	top := root(func(value any) { exiting <- value.(*Failure); <-releaseExit }, func(any, []byte) {
		close(reported)
		<-releaseReport
	})
	releaseStop := holdRootStop(t, top)
	go func() {
		_, err := Start[initializing](top, &initHooks{
			init: func(p *initializing) error {
				_, err := Start[initializing](p.UnsafeChildren(), &initHooks{
					init: func(*initializing) error { return nil },
					run: func(_ *initializing, ctx context.Context) {
						if ctx.Err() == nil {
							t.Error("late child started with a live context")
						}
						close(childCleanup)
						<-releaseChild
					},
				})
				close(initReady)
				<-releaseInit
				return err
			},
			run: func(_ *initializing, ctx context.Context) {
				if ctx.Err() == nil {
					t.Error("late Run started with a live context")
				}
				if mode == "Goexit" {
					runtime.Goexit()
				}
				panic("late failure")
			},
		})
		started <- err
	}()
	<-initReady
	stopped := top.Stop()
	if state == "stopped" {
		close(releaseStop)
		<-stopped
	}
	close(releaseInit)
	<-childCleanup
	synctest.Wait()
	select {
	case <-reported:
		t.Error("parent Run began before staged child cleanup ended")
	default:
	}
	close(releaseChild)
	<-reported
	if state == "stopping" {
		close(releaseStop)
		synctest.Wait()
		select {
		case <-stopped:
			t.Error("Stop missed the unregistered service failure")
		default:
		}
	}
	synctest.Wait()
	if len(exiting) != 0 {
		t.Error("terminal action ran before reporting ended")
	}
	close(releaseReport)
	failure := <-exiting
	synctest.Wait()
	if state == "stopped" && len(started) != 0 {
		t.Error("late Start returned before its terminal action")
	}
	close(releaseExit)
	<-stopped
	if err := <-started; err != nil {
		t.Fatal(err)
	}
	if mode == "Goexit" && failure.Value != errGoexit || mode == "panic" && failure.Value != "late failure" {
		t.Fatalf("terminal failure = %v", failure)
	}
}

// holdRootStop keeps one registered service in cleanup until the returned channel closes.
func holdRootStop(t *testing.T, top *Group) chan struct{} {
	t.Helper()
	release := make(chan struct{})
	_, err := Start[initializing](top, &initHooks{
		init: func(*initializing) error { return nil },
		run:  func(_ *initializing, ctx context.Context) { <-ctx.Done(); <-release },
	})
	if err != nil {
		t.Fatal(err)
	}
	return release
}
