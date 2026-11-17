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

// runCause supplies a typed original error for errors.As.
type runCause struct{ message string }

// Error returns the original panic text.
func (e *runCause) Error() string { return e.message }

// TestFirstRunFailure preserves the first failure without another Run or delay.
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

// checkFirstRunFailure checks the error, stack, reports, and Run count.
func checkFirstRunFailure(t *testing.T, mode string, reporting bool) {
	t.Helper()
	var cause any = "ledger: balance underflow"
	if mode == "error" {
		cause = &runCause{message: cause.(string)}
	} else if mode == "Goexit" {
		cause = errGoexit
	}
	reports := make(chan any, 2)
	var runStack []byte
	report := func(value any, stack []byte) {
		runStack = stack
		reports <- value
	}
	if !reporting {
		report = nil
	}
	terminal := make(chan any, 1)
	top := root(func(value any) { terminal <- value }, report)
	parent := StartSupervisor(StartSupervisor(top))
	trigger := make(chan struct{})
	inits, runs := 0, 0
	ref, err := Start[initializing](parent, &initHooks{
		init: func(*initializing) error { inits++; return nil },
		run: func(*initializing, context.Context) {
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
	failure := checkFirstFailureDetails(t, value, cause)
	if elapsed := time.Since(at); elapsed != 0 || inits != 1 || runs != 1 {
		t.Errorf("elapsed=%s Init=%d Run=%d; want 0s, 1, 1", elapsed, inits, runs)
	}
	if ref.Borrow() != original {
		t.Error("stopped reference changed its service value")
	}
	if reporting {
		if len(reports) != 1 || <-reports != failure {
			t.Error("expected one Failure report")
		}
		if !bytes.Equal(runStack, failure.Stack) {
			t.Error("terminal Failure changed the Run stack")
		}
	}
}

// checkFirstFailureDetails checks the original value, error type, and Run stack.
func checkFirstFailureDetails(t *testing.T, value, cause any) *Failure {
	t.Helper()
	failure, ok := value.(*Failure)
	if !ok {
		t.Fatalf("terminal value = %T; want *Failure", value)
	}
	if failure.Service != "supervise.initializing" || failure.Value != cause {
		t.Errorf("failure = %#v", failure)
	}
	if !bytes.Contains(failure.Stack, []byte("checkFirstRunFailure.func")) {
		t.Errorf("original Run stack was lost:\n%s", failure.Stack)
	}
	if err, ok := cause.(error); ok && !errors.Is(failure, err) {
		t.Errorf("original error was lost: %v", failure)
	}
	if _, typed := cause.(*runCause); typed {
		var originalError *runCause
		if !errors.As(failure, &originalError) || originalError != cause {
			t.Errorf("original error type was lost: %v", failure)
		}
	}
	return failure
}
