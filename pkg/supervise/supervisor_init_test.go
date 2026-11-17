package supervise_test

import (
	"context"
	"errors"
	"testing"
	"testing/synctest"

	"github.com/lucasavila00/immutago/pkg/supervise"
)

// poolEvents records each worker's start and completion.
type poolEvents struct {
	started chan string
	stopped chan string
}

// dynamicPool retains the supervisor supplied by the runtime.
type dynamicPool struct {
	supervise.Service[*poolEvents]
	supervisor *supervise.Supervisor
}

// Init saves the supervisor before the pool becomes available to callers.
func (p *dynamicPool) Init(supervisor *supervise.Supervisor) error {
	p.supervisor = supervisor
	_, err := p.Add("initial")
	return err
}

// Add starts a worker under the pool's supervisor.
func (p *dynamicPool) Add(name string) (supervise.Ref[poolWorker], error) {
	return supervise.Start[poolWorker](p.supervisor, poolWorkerArgs{name, p.Args()})
}

// Run keeps the pool alive until its workers have stopped.
func (p *dynamicPool) Run(ctx context.Context) { <-ctx.Done() }

// poolWorkerArgs supplies one worker's name and observation channels.
type poolWorkerArgs struct {
	name   string
	events *poolEvents
}

// poolWorker reports startup and shutdown without external resources.
type poolWorker struct {
	supervise.Service[poolWorkerArgs]
}

// Run reports both lifecycle boundaries.
func (p *poolWorker) Run(ctx context.Context) {
	args := p.Args()
	args.events.started <- args.name
	<-ctx.Done()
	args.events.stopped <- args.name
}

// TestInitSupervisorOwnsDynamicServices checks startup, selective stop, and late starts through the public API.
func TestInitSupervisorOwnsDynamicServices(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		root := supervise.NewRoot(nil)
		defer func() { <-root.Stop() }()
		events := &poolEvents{make(chan string, 3), make(chan string, 3)}
		ref, err := supervise.Start[dynamicPool](root, events)
		if err != nil {
			t.Fatal(err)
		}
		pool := ref.Borrow()
		expectPoolEvent(t, events.started, "initial")
		if _, err := pool.Add("dynamic"); err != nil {
			t.Fatal(err)
		}
		expectPoolEvent(t, events.started, "dynamic")
		selected := supervise.StartSupervisor(pool.supervisor)
		if _, err := supervise.Start[poolWorker](selected, poolWorkerArgs{"selected", events}); err != nil {
			t.Fatal(err)
		}
		expectPoolEvent(t, events.started, "selected")
		<-selected.Stop()
		expectPoolEvent(t, events.stopped, "selected")
		select {
		case name := <-events.stopped:
			t.Fatalf("selective stop also stopped %q", name)
		default:
		}
		<-root.Stop()
		for _, want := range []string{"dynamic", "initial"} {
			if got := <-events.stopped; got != want {
				t.Fatalf("shutdown order = %q, want %q", got, want)
			}
		}
		late, err := pool.Add("late")
		if !errors.Is(err, context.Canceled) || !late.IsZero() {
			t.Fatalf("late start = %v, zero = %v", err, late.IsZero())
		}
	})
}

// expectPoolEvent checks one lifecycle event in order.
func expectPoolEvent(t *testing.T, events <-chan string, want string) {
	t.Helper()
	if got := <-events; got != want {
		t.Fatalf("event = %q, want %q", got, want)
	}
}
