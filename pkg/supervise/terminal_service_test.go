package supervise

import (
	"bytes"
	"context"
	"errors"
	"runtime"
	"testing"
	"testing/synctest"
	"time"
)

// terminalHooks controls the first and only attempt of a terminal service.
type terminalHooks struct {
	init func(*terminalWorker) error
	run  func(*terminalWorker, context.Context)
}

// terminalWorker exposes setup and background work to the runtime tests.
type terminalWorker struct {
	TerminalService[*terminalHooks]
}

// Init calls the test's setup function.
func (p *terminalWorker) Init() error { return p.Args().init(p) }

// Run calls the test's background function.
func (p *terminalWorker) Run(ctx context.Context) { p.Args().run(p, ctx) }

// terminalCause supplies a typed original error for errors.As.
type terminalCause struct{ message string }

// Error returns the original panic text.
func (e *terminalCause) Error() string { return e.message }

// TestTerminalServiceFailure preserves the first failure without a retry or delay.
func TestTerminalServiceFailure(t *testing.T) {
	for _, mode := range []string{"panic", "error", "Goexit"} {
		for _, reporting := range []bool{false, true} {
			name := mode + "/silent"
			if reporting {
				name = mode + "/report"
			}
			t.Run(name, func(t *testing.T) {
				synctest.Test(t, func(t *testing.T) { checkTerminalServiceFailure(t, mode, reporting) })
			})
		}
	}
}

// checkTerminalServiceFailure checks the error, stack, reports, and attempt count.
func checkTerminalServiceFailure(t *testing.T, mode string, reporting bool) {
	t.Helper()
	var cause any = "ledger: balance underflow"
	if mode == "error" {
		cause = &terminalCause{message: cause.(string)}
	} else if mode == "Goexit" {
		cause = errGoexit
	}
	reports := make(chan any, 2)
	var attemptStack []byte
	report := func(value any, stack []byte) {
		if _, terminal := value.(*Failure); !terminal {
			attemptStack = stack
		}
		reports <- value
	}
	if !reporting {
		report = nil
	}
	terminal := make(chan any, 1)
	top := root(func(value any) { terminal <- value }, report)
	parent := StartGroup(StartGroup(top))
	trigger := make(chan struct{})
	inits, runs := 0, 0
	ref, err := Start[terminalWorker](parent, &terminalHooks{
		init: func(*terminalWorker) error { inits++; return nil },
		run: func(*terminalWorker, context.Context) {
			runs++
			<-trigger
			if mode == "Goexit" {
				runtime.Goexit()
			}
			panic(cause)
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	original := ref.Borrow()
	at := time.Now()
	close(trigger)
	value := <-terminal
	<-top.Stop()
	failure, ok := value.(*Failure)
	if !ok {
		t.Fatalf("terminal value = %T; want *Failure", value)
	}
	if failure.Service != "supervise.terminalWorker" || failure.Value != cause {
		t.Errorf("failure = %#v", failure)
	}
	if !bytes.Contains(failure.Stack, []byte("checkTerminalServiceFailure.func")) {
		t.Errorf("original Run stack was lost:\n%s", failure.Stack)
	}
	if err, ok := cause.(error); ok && !errors.Is(failure, err) {
		t.Errorf("original error was lost: %v", failure)
	}
	if mode == "error" {
		var originalError *terminalCause
		if !errors.As(failure, &originalError) || originalError != cause {
			t.Errorf("original error type was lost: %v", failure)
		}
	}
	if elapsed := time.Since(at); elapsed != 0 || inits != 1 || runs != 1 {
		t.Errorf("elapsed=%s Init=%d Run=%d; want 0s, 1, 1", elapsed, inits, runs)
	}
	if ref.Borrow() != original {
		t.Error("stopped reference changed its service value")
	}
	if reporting {
		if len(reports) != 2 || <-reports != cause || <-reports != failure {
			t.Error("expected one attempt report followed by the terminal Failure")
		}
		if !bytes.Equal(attemptStack, failure.Stack) {
			t.Error("terminal Failure changed the attempt stack")
		}
	}
}
