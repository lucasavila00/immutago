package supervise

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"sync"
	"testing"
	"testing/synctest"
	"time"
)

// failureLeaf fails with the error supplied at startup.
type failureLeaf struct{ Service[error] }

// Run fails on each attempt.
func (s *failureLeaf) Run(context.Context) { panic(s.Args()) }

// failureParent owns a child that always fails.
type failureParent struct{ Service[error] }

// Init creates the child before its parent starts running.
func (s *failureParent) Init() error {
	_, err := Start[failureLeaf](s.Children(), s.Args())
	return err
}

// Run waits for shutdown.
func (*failureParent) Run(ctx context.Context) { <-ctx.Done() }

// TestFailureOrigin checks that ownership depth does not change failure details.
func TestFailureOrigin(t *testing.T) {
	for _, shape := range []string{"direct", "groups", "service"} {
		for _, reporting := range []bool{false, true} {
			name := shape + "/silent"
			if reporting {
				name = shape + "/report"
			}
			t.Run(name, func(t *testing.T) {
				synctest.Test(t, func(t *testing.T) { checkFailureOrigin(t, shape, reporting) })
			})
		}
	}
}

// checkFailureOrigin checks the terminal record and unchanged attempt reports.
func checkFailureOrigin(t *testing.T, shape string, reporting bool) {
	t.Helper()
	cause := errors.New("model load failed")
	terminal := make(chan any, 1)
	var mu sync.Mutex
	var reports []any
	report := func(value any, stack []byte) {
		if !bytes.Contains(stack, []byte("failureLeaf).Run")) {
			t.Errorf("report lost the leaf stack:\n%s", stack)
		}
		mu.Lock()
		defer mu.Unlock()
		reports = append(reports, value)
	}
	if !reporting {
		report = nil
	}
	top := root(func(value any) { terminal <- value }, report)
	parent := top
	if shape == "groups" {
		for range 4 {
			parent = StartGroup(parent)
		}
	}
	var err error
	if shape == "service" {
		_, err = Start[failureParent](parent, cause)
	} else {
		_, err = Start[failureLeaf](parent, cause)
	}
	if err != nil {
		t.Fatal(err)
	}
	var value any
	select {
	case value = <-terminal:
	case <-time.After(time.Minute):
		<-top.Stop()
		t.Fatal("terminal failure did not reach root within the leaf retry budget")
	}
	<-top.Stop()
	failure, ok := value.(*Failure)
	if !ok {
		t.Fatalf("terminal failure = %T, want *Failure", value)
	}
	if failure.Service != "supervise.failureLeaf" || failure.Value != cause {
		t.Errorf("failure = %#v", failure)
	}
	if !errors.Is(failure, cause) || !bytes.Contains(failure.Stack, []byte("failureLeaf).Run")) {
		t.Errorf("original error or stack lost: %v\n%s", failure, failure.Stack)
	}
	mu.Lock()
	defer mu.Unlock()
	if reporting && len(reports) != 6 {
		t.Errorf("reports = %d, want six leaf failures with no parent reports", len(reports))
	}
	for _, reported := range reports {
		if reported != cause {
			t.Errorf("attempt report changed the original panic: %v", reported)
		}
	}
}

// TestFailureError checks the public error behavior for both error and non-error panics.
func TestFailureError(t *testing.T) {
	for _, value := range []any{errors.New("model load failed"), 42} {
		failure := &Failure{Service: "worker.Model", Value: value}
		var found *Failure
		if !errors.As(failure, &found) || found != failure || !strings.Contains(failure.Error(), "worker.Model") {
			t.Errorf("failure identity or text lost: %v", failure)
		}
		if cause, ok := value.(error); ok {
			if !errors.Is(failure, cause) {
				t.Errorf("original error lost: %v", failure)
			}
		} else if failure.Unwrap() != nil || !strings.Contains(failure.Error(), "42") {
			t.Errorf("non-error panic changed: %v", failure)
		}
	}
}
