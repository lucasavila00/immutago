package supervise

import (
	"context"
	"runtime"
	"testing"
	"testing/synctest"
)

// TestChildFailureAfterParentReturn holds a child failure until its parent starts child cleanup.
func TestChildFailureAfterParentReturn(t *testing.T) {
	for _, where := range []string{"defer", "report"} {
		for _, cause := range []string{"panic", "Goexit"} {
			t.Run(where+"/"+cause, func(t *testing.T) {
				synctest.Test(t, func(t *testing.T) { checkChildFailureAfterParentReturn(t, where, cause) })
			})
		}
	}
}

// checkChildFailureAfterParentReturn checks one failure and one observation delay.
func checkChildFailureAfterParentReturn(t *testing.T, where, cause string) {
	t.Helper()
	notified, held, release := make(chan struct{}), make(chan struct{}), make(chan struct{})
	exited := make(chan any, 1)
	var report func(any, []byte)
	if where == "report" {
		report = func(value any, _ []byte) {
			close(held)
			<-release
		}
	}
	top := root(func(value any) { exited <- value }, report)
	var children *Supervisor
	_, err := Start[initializing](top, &initHooks{
		init: func(p *initializing) error {
			children = p.supervisor
			_, err := Start[initializing](children, &initHooks{
				init: func(*initializing) error { return nil },
				run: func(*initializing, context.Context) {
					defer func() {
						close(notified)
						if where == "defer" {
							close(held)
							<-release
						}
					}()
					if cause == "Goexit" {
						runtime.Goexit()
					}
					panic("child failure")
				},
			})
			return err
		},
		run: func(*initializing, context.Context) { <-notified },
	})
	if err != nil {
		t.Fatal(err)
	}
	<-held
	synctest.Wait()
	if children.admission.Err() == nil {
		t.Fatal("parent did not start child cleanup")
	}
	close(release)
	synctest.Wait()
	select {
	case <-exited:
	default:
		t.Error("child failure was lost after normal parent return")
	}
	<-top.Stop()
}
