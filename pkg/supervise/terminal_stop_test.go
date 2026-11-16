package supervise

import (
	"context"
	"slices"
	"testing"
	"testing/synctest"
)

// TestTerminalFailureStopOrder orders the attempt report, Stop, and terminal selection.
func TestTerminalFailureStopOrder(t *testing.T) {
	for _, mode := range []string{"root first", "subtree first", "unrelated first", "failure first"} {
		t.Run(mode, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) { checkTerminalFailureStopOrder(t, mode) })
		})
	}
}

// checkTerminalFailureStopOrder holds the attempt report before the terminal decision.
func checkTerminalFailureStopOrder(t *testing.T, mode string) {
	t.Helper()
	attempt, release := make(chan struct{}), make(chan struct{})
	terminal, exited := make(chan any, 1), make(chan any, 1)
	top := root(func(value any) { exited <- value }, func(value any, _ []byte) {
		if _, failed := value.(*Failure); failed {
			terminal <- value
			return
		}
		close(attempt)
		<-release
	})
	parent := StartGroup(top)
	_, err := Start[terminalWorker](parent, &terminalHooks{
		init: func(*terminalWorker) error { return nil },
		run:  func(*terminalWorker, context.Context) { panic("first failure") },
	})
	if err != nil {
		t.Fatal(err)
	}
	<-attempt
	var stopped <-chan struct{}
	switch mode {
	case "root first":
		stopped = top.Stop()
	case "subtree first":
		stopped = parent.Stop()
	case "unrelated first":
		<-StartGroup(top).Stop()
	}
	close(release)
	if stopped != nil {
		<-stopped
		synctest.Wait()
		if len(terminal) != 0 || len(exited) != 0 {
			t.Error("an earlier Stop did not suppress the terminal failure")
		}
	} else {
		failure := (<-terminal).(*Failure)
		<-top.Stop()
		if value := <-exited; value != failure || failure.Value != "first failure" {
			t.Errorf("terminal failure = %v, exit = %v", failure, value)
		}
	}
	<-top.Stop()
}

// TestTerminalCleanupBeforeExit checks nested service failure and ordered cleanup.
func TestTerminalCleanupBeforeExit(t *testing.T) {
	synctest.Test(t, checkTerminalCleanupBeforeExit)
}

// checkTerminalCleanupBeforeExit holds one child open until the root reports failure.
func checkTerminalCleanupBeforeExit(t *testing.T) {
	t.Helper()
	reported, exited := make(chan *Failure, 1), make(chan any, 1)
	top := root(func(value any) { exited <- value }, func(value any, _ []byte) {
		if failure, ok := value.(*Failure); ok {
			reported <- failure
		}
	})
	trigger, blocked, release := make(chan struct{}), make(chan struct{}), make(chan struct{})
	var order []string
	child := func(name string) *initHooks {
		return &initHooks{
			init: func(*initializing) error { return nil },
			run: func(_ *initializing, ctx context.Context) {
				<-ctx.Done()
				if name == "second" {
					close(blocked)
					<-release
				}
				order = append(order, name)
			},
		}
	}
	if _, err := Start[initializing](top, child("peer")); err != nil {
		t.Fatal(err)
	}
	inits := 0
	parent := child("parent")
	parent.init = func(p *initializing) error {
		inits++
		_, err := Start[terminalWorker](StartGroup(p.Children()), &terminalHooks{
			init: func(p *terminalWorker) error {
				for _, name := range []string{"first", "second"} {
					if _, err := Start[initializing](p.Children(), child(name)); err != nil {
						return err
					}
				}
				return nil
			},
			run: func(*terminalWorker, context.Context) { <-trigger; panic("nested failure") },
		})
		return err
	}
	if _, err := Start[initializing](top, parent); err != nil {
		t.Fatal(err)
	}
	synctest.Wait()
	close(trigger)
	failure := <-reported
	<-blocked
	synctest.Wait()
	if len(exited) != 0 {
		t.Error("root exited before its children stopped")
	}
	close(release)
	if value := <-exited; value != failure || failure.Value != "nested failure" {
		t.Errorf("terminal failure = %v, exit = %v", failure, value)
	}
	<-top.Stop()
	if inits != 1 || !slices.Equal(order, []string{"second", "first", "parent", "peer"}) {
		t.Errorf("parent Init calls = %d, cleanup order = %v", inits, order)
	}
}
