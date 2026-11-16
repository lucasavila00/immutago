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

// failureHooks controls the single attempt of a service.
type failureHooks struct {
	init func(*failureWorker) error
	run  func(*failureWorker, context.Context)
}

// failureWorker exposes setup and background work to the runtime tests.
type failureWorker struct {
	Service[*failureHooks]
}

// Init calls the test's setup function.
func (p *failureWorker) Init() error { return p.Args().init(p) }

// Run calls the test's background function.
func (p *failureWorker) Run(ctx context.Context) { p.Args().run(p, ctx) }

// runCause supplies a typed original error for errors.As.
type runCause struct{ message string }

// Error returns the original panic text.
func (e *runCause) Error() string { return e.message }

// TestFirstRunFailure preserves the first failure without another attempt or delay.
func TestFirstRunFailure(t *testing.T) {
	for _, mode := range []string{"panic", "error", "Goexit"} {
		for _, reporting := range []bool{false, true} {
			name := mode + "/silent"
			if reporting {
				name = mode + "/report"
			}
			t.Run(name, func(t *testing.T) {
				synctest.Test(t, func(t *testing.T) { checkFirstRunFailure(t, mode, reporting) })
			})
		}
	}
}

// checkFirstRunFailure checks the error, stack, reports, and attempt count.
func checkFirstRunFailure(t *testing.T, mode string, reporting bool) {
	t.Helper()
	var cause any = "ledger: balance underflow"
	if mode == "error" {
		cause = &runCause{message: cause.(string)}
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
	ref, err := Start[failureWorker](parent, &failureHooks{
		init: func(*failureWorker) error { inits++; return nil },
		run: func(*failureWorker, context.Context) {
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
	if failure.Service != "supervise.failureWorker" || failure.Value != cause {
		t.Errorf("failure = %#v", failure)
	}
	if !bytes.Contains(failure.Stack, []byte("checkFirstRunFailure.func")) {
		t.Errorf("original Run stack was lost:\n%s", failure.Stack)
	}
	if err, ok := cause.(error); ok && !errors.Is(failure, err) {
		t.Errorf("original error was lost: %v", failure)
	}
	if mode == "error" {
		var originalError *runCause
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
