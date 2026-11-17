package supervise

import (
	"context"
	"testing"
	"testing/synctest"
)

// TestClosedStartRejectsUserCode checks root, subtree, and inherited admission closure.
func TestClosedStartRejectsUserCode(t *testing.T) {
	for _, owner := range []string{"root", "nested", "ancestor"} {
		for _, state := range []string{"stopping", "stopped"} {
			t.Run(owner+"/"+state, func(t *testing.T) {
				synctest.Test(t, func(t *testing.T) { checkClosedStart(t, owner, state) })
			})
		}
	}
}

// checkClosedStart holds cleanup so rejection is tested before and after completion.
func checkClosedStart(t *testing.T, owner, state string) {
	t.Helper()
	top := NewRoot(nil)
	parent := top
	if owner != "root" {
		parent = StartSupervisor(top)
	}
	cleaning, release := make(chan struct{}), make(chan struct{})
	_, err := Start[initializing](parent, &initHooks{
		init: func(*initializing) error { return nil },
		run:  func(_ *initializing, ctx context.Context) { <-ctx.Done(); close(cleaning); <-release },
	})
	if err != nil {
		t.Fatal(err)
	}
	stop := parent
	if owner == "ancestor" {
		stop = top
	}
	done := stop.Stop()
	<-cleaning
	if state == "stopped" {
		close(release)
		<-done
	}
	for _, group := range []*Supervisor{parent, StartSupervisor(parent)} {
		inits, runs := 0, 0
		ref, err := Start[initializing](group, &initHooks{
			init: func(*initializing) error { inits++; return nil },
			run:  func(*initializing, context.Context) { runs++ },
		})
		if err != context.Canceled || !ref.IsZero() || inits != 0 || runs != 0 {
			t.Errorf("closed Start: error=%v zero=%t Init=%d Run=%d", err, ref.IsZero(), inits, runs)
		}
	}
	if state == "stopping" {
		close(release)
		<-done
	}
	<-top.Stop()
}
