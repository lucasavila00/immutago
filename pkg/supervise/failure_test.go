package supervise

import (
	"bytes"
	"context"
	"errors"
	"slices"
	"strings"
	"sync"
	"testing"
	"testing/synctest"
)

// failureLeaf fails with the error supplied at startup.
type failureLeaf struct{ Service[error] }

// Run fails on each attempt.
func (s *failureLeaf) Run(context.Context) { panic(s.Args()) }

// failureTreeArgs sets the parent depth above one failing leaf.
type failureTreeArgs struct {
	depth int
	cause error
}

// failureParent owns another parent or the failing leaf.
type failureParent struct{ Service[failureTreeArgs] }

// Init creates the child before its parent starts running.
func (s *failureParent) Init() error {
	args := s.Args()
	if args.depth > 0 {
		args.depth--
		_, err := Start[failureParent](s.UnsafeChildren(), args)
		return err
	}
	_, err := Start[failureLeaf](s.UnsafeChildren(), args.cause)
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

// checkFailureOrigin checks the terminal record and one report per failure.
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
		_, err = Start[failureParent](parent, failureTreeArgs{depth: 3, cause: cause})
	} else {
		_, err = Start[failureLeaf](parent, cause)
	}
	if err != nil {
		t.Fatal(err)
	}
	value := <-terminal
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
	var want []any
	if reporting {
		want = []any{failure}
	}
	if !slices.Equal(reports, want) {
		t.Errorf("reports = %v, want %v", reports, want)
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
