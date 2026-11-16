package supervise

import (
	"context"
	"sync/atomic"
	"testing"
	"testing/synctest"
	"time"
	"unsafe"
)

// selectionCalls counts attempts across fresh service values.
type selectionCalls struct{ runs atomic.Int32 }

// run fails once, then waits for shutdown after a retry.
func (s *selectionCalls) run(ctx context.Context) {
	if s.runs.Add(1) == 1 {
		panic("first attempt")
	}
	<-ctx.Done()
}

// selectionRetry uses the retrying service base.
type selectionRetry struct{ Service[*selectionCalls] }

// Run calls the common test operation.
func (s *selectionRetry) Run(ctx context.Context) { s.Args().run(ctx) }

// selectionTerminal uses the terminal base through an alias.
type selectionTerminalBase = TerminalService[*selectionCalls]

// selectionTerminal declares the service operation.
type selectionTerminal struct{ selectionTerminalBase }

// Run calls the common test operation.
func (s *selectionTerminal) Run(ctx context.Context) { s.Args().run(ctx) }

// selectionRetryAlias preserves the retrying service identity.
type selectionRetryAlias = selectionRetry

// selectionTerminalAlias preserves the terminal service identity.
type selectionTerminalAlias = selectionTerminal

// selectionNestedRetry adds a value path to the retrying base.
type selectionNestedRetry struct{ selectionRetryAlias }

// selectionNestedTerminal adds a value path to the terminal base.
type selectionNestedTerminal struct{ selectionTerminalAlias }

// selectionDefinedRetry retains the embedded base in a new outer type.
type selectionDefinedRetry selectionRetry

// Run calls the operation for the new outer type.
func (s *selectionDefinedRetry) Run(ctx context.Context) { s.Args().run(ctx) }

// selectionDefinedTerminal retains the embedded terminal base in a new outer type.
type selectionDefinedTerminal selectionTerminal

// Run calls the operation for the new outer type.
func (s *selectionDefinedTerminal) Run(ctx context.Context) { s.Args().run(ctx) }

// selectionMixedRetry selects the shallower retrying base.
type selectionMixedRetry struct {
	Service[*selectionCalls]
	selectionTerminal
}

// Run uses the arguments from the selected base.
func (s *selectionMixedRetry) Run(ctx context.Context) { s.Args().run(ctx) }

// selectionMixedTerminal selects the shallower terminal base.
type selectionMixedTerminal struct {
	TerminalService[*selectionCalls]
	selectionRetry
}

// Run uses the arguments from the selected base.
func (s *selectionMixedTerminal) Run(ctx context.Context) { s.Args().run(ctx) }

// TestServiceBaseSelection preserves policy across all supported value paths.
func TestServiceBaseSelection(t *testing.T) {
	for _, test := range []struct {
		name     string
		terminal bool
		check    func(*testing.T, bool)
	}{
		{"retry", false, checkServiceBaseSelection[selectionRetry]},
		{"retry alias", false, checkServiceBaseSelection[selectionRetryAlias]},
		{"retry nested", false, checkServiceBaseSelection[selectionNestedRetry]},
		{"retry defined", false, checkServiceBaseSelection[selectionDefinedRetry]},
		{"retry anonymous", false, checkServiceBaseSelection[struct{ selectionRetry }]},
		{"retry mixed", false, checkServiceBaseSelection[selectionMixedRetry]},
		{"terminal", true, checkServiceBaseSelection[selectionTerminal]},
		{"terminal alias", true, checkServiceBaseSelection[selectionTerminalAlias]},
		{"terminal nested", true, checkServiceBaseSelection[selectionNestedTerminal]},
		{"terminal defined", true, checkServiceBaseSelection[selectionDefinedTerminal]},
		{"terminal anonymous", true, checkServiceBaseSelection[struct{ selectionTerminal }]},
		{"terminal mixed", true, checkServiceBaseSelection[selectionMixedTerminal]},
	} {
		t.Run(test.name, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) { test.check(t, test.terminal) })
		})
	}
}

// checkServiceBaseSelection observes the first failure and an optional replacement.
func checkServiceBaseSelection[T any, P interface {
	*T
	Init() error
	Run(context.Context)
	bind(*selectionCalls, *Group) (*service[*selectionCalls], int)
}](t *testing.T, terminal bool) {
	t.Helper()
	exited := make(chan any, 1)
	top := root(func(value any) { exited <- value }, func(any, []byte) {})
	calls := new(selectionCalls)
	ref, err := Start[T, *selectionCalls, P](top, calls)
	if err != nil {
		t.Fatal(err)
	}
	first := ref.Borrow()
	synctest.Wait()
	if calls.runs.Load() != 1 || (len(exited) == 1) != terminal {
		t.Fatalf("first failure: Run calls = %d, terminal failures = %d", calls.runs.Load(), len(exited))
	}
	if !terminal {
		time.Sleep(time.Second)
		synctest.Wait()
		if calls.runs.Load() != 2 || ref.Borrow() == first {
			t.Error("retry did not publish a fresh service after one second")
		}
	}
	<-top.Stop()
}

// TestServiceBaseRepresentation checks that each public base adds no runtime storage.
func TestServiceBaseRepresentation(t *testing.T) {
	want := unsafe.Sizeof(service[NoArgs]{})
	if unsafe.Sizeof(Service[NoArgs]{}) != want || unsafe.Sizeof(TerminalService[NoArgs]{}) != want {
		t.Fatal("a service base adds storage to the shared runtime state")
	}
}

// TestMixedServiceBaseState binds only the base that selects the retry behavior.
func TestMixedServiceBaseState(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		top := root(func(any) {}, func(any, []byte) {})
		retryCalls, terminalCalls := new(selectionCalls), new(selectionCalls)
		retry, err := Start[selectionMixedRetry](top, retryCalls)
		if err != nil {
			t.Fatal(err)
		}
		terminal, err := Start[selectionMixedTerminal](top, terminalCalls)
		if err != nil {
			t.Fatal(err)
		}
		if retry.Borrow().Args() != retryCalls || retry.Borrow().selectionTerminal.Args() != nil {
			t.Error("retrying service bound the hidden terminal base")
		}
		if terminal.Borrow().Args() != terminalCalls || terminal.Borrow().selectionRetry.Args() != nil {
			t.Error("terminal service bound the hidden retrying base")
		}
		<-top.Stop()
	})
}
