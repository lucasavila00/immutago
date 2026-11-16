package supervise

import (
	"context"
	"errors"
	"runtime"
	"testing"
	"testing/synctest"
)

// TestTerminalInitFailure keeps initial setup failures in the Start caller.
func TestTerminalInitFailure(t *testing.T) {
	for _, mode := range []string{"error", "panic", "Goexit"} {
		t.Run(mode, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) { checkTerminalInitFailure(t, mode) })
		})
	}
}

// checkTerminalInitFailure checks staged child cleanup without a report or Run.
func checkTerminalInitFailure(t *testing.T, mode string) {
	t.Helper()
	top := root(func(any) { t.Error("initial Init failure stopped the root") },
		func(any, []byte) { t.Error("initial Init failure was reported") })
	cause := errors.New("initial setup")
	var parent *terminalWorker
	inits, childRuns := 0, 0
	hooks := &terminalHooks{
		init: func(p *terminalWorker) error {
			inits++
			parent = p
			if _, err := Start[initializing](p.Children(), &initHooks{
				init: func(*initializing) error { return nil },
				run: func(_ *initializing, ctx context.Context) {
					childRuns++
					if ctx.Err() == nil {
						t.Error("staged child received an active context")
					}
				},
			}); err != nil {
				return err
			}
			switch mode {
			case "panic":
				panic(cause)
			case "Goexit":
				runtime.Goexit()
			}
			return cause
		},
		run: func(*terminalWorker, context.Context) { t.Error("Run followed a failed Init") },
	}
	var ref Ref[terminalWorker]
	var err error
	var recovered any
	finished := make(chan struct{})
	go func() {
		defer close(finished)
		defer func() { recovered = recover() }()
		ref, err = Start[terminalWorker](top, hooks)
	}()
	<-finished
	if mode == "error" && err != cause || mode == "panic" && recovered != cause {
		t.Errorf("error = %v, panic = %v", err, recovered)
	}
	if mode == "Goexit" && (err != nil || recovered != nil) {
		t.Errorf("Goexit became an error or panic: %v, %v", err, recovered)
	}
	if !ref.IsZero() || inits != 1 || childRuns != 1 {
		t.Errorf("reference is zero = %t, Init calls = %d, child Run calls = %d", ref.IsZero(), inits, childRuns)
	}
	select {
	case <-parent.Children().stopped:
	default:
		t.Error("failed Init left its children active")
	}
	<-top.Stop()
}

// TestTerminalInitAfterStop gives late successful setup one cancelled cleanup Run.
func TestTerminalInitAfterStop(t *testing.T) {
	for _, before := range []bool{false, true} {
		synctest.Test(t, func(t *testing.T) {
			top := NewRoot(nil)
			entered, release, finished := make(chan struct{}), make(chan struct{}), make(chan struct{})
			if before {
				<-top.Stop()
			}
			var candidate *terminalWorker
			var ref Ref[terminalWorker]
			inits, runs := 0, 0
			go func() {
				defer close(finished)
				var err error
				ref, err = Start[terminalWorker](top, &terminalHooks{
					init: func(p *terminalWorker) error {
						inits++
						candidate = p
						p.Children()
						close(entered)
						<-release
						return nil
					},
					run: func(_ *terminalWorker, ctx context.Context) {
						runs++
						if ctx.Err() == nil {
							t.Error("late setup ran with an active context")
						}
					},
				})
				if err != nil {
					t.Error(err)
				}
			}()
			<-entered
			<-top.Stop()
			close(release)
			<-finished
			if inits != 1 || runs != 1 || ref.IsZero() || ref.Borrow() != candidate {
				t.Fatalf("Init calls = %d, Run calls = %d, reference = %v", inits, runs, ref)
			}
			select {
			case <-candidate.Children().stopped:
			default:
				t.Error("late setup left its children active")
			}
		})
	}
}
