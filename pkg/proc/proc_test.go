package proc

import (
	"context"
	"slices"
	"sync"
	"sync/atomic"
	"testing"
	"testing/synctest"
	"time"

	"github.com/lucasavila00/immutago/pkg/im"
)

var (
	eventsMu sync.Mutex
	events   []string
)

// record appends one event for the current test.
func record(event string) {
	eventsMu.Lock()
	defer eventsMu.Unlock()
	events = append(events, event)
}

// takeEvents returns and clears the recorded events.
func takeEvents() []string {
	eventsMu.Lock()
	defer eventsMu.Unlock()
	taken := events
	events = nil
	return taken
}

// track records the start and the stop of a process that runs until ctx ends.
func track(ctx context.Context, name string) {
	record("start " + name)
	<-ctx.Done()
	record("stop " + name)
}

// first is a process that runs until it stops.
type first struct{ Process[None] }

// Run records the start and the stop.
func (*first) Run(ctx context.Context) { track(ctx, "first") }

// second is a process that runs until it stops.
type second struct{ Process[None] }

// Run records the start and the stop.
func (*second) Run(ctx context.Context) { track(ctx, "second") }

// third is a process that runs until it stops.
type third struct{ Process[None] }

// Run records the start and the stop.
func (*third) Run(ctx context.Context) { track(ctx, "third") }

// counter is a process with a mutex-protected count.
type counter struct {
	Process[None]
	mu    sync.Mutex
	count int
}

// Run waits until the process stops.
func (c *counter) Run(ctx context.Context) { <-ctx.Done() }

// Add increases the count and returns the new count.
func (c *counter) Add() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.count++
	return c.count
}

var failures atomic.Int64

// failing is a process that panics on each run.
type failing struct{ Process[None] }

// Run records the run time and panics with the number of failures.
func (*failing) Run(context.Context) {
	record(time.Now().UTC().Format(time.TimeOnly))
	panic(failures.Add(1))
}

// label is the argument of a restarted process.
type label struct {
	im.Shared
	Name string
}

// snapshot is a value that a restarted process publishes.
type snapshot struct{ im.Scoped }

// restarted is a process that panics on its first run.
type restarted struct {
	Process[label]
	runs   int
	latest im.Owner[snapshot]
}

// Run records the arguments and the state of the value, and panics on the first failure.
func (r *restarted) Run(ctx context.Context) {
	r.runs++
	record(r.Args().Name)
	if r.runs != 1 || r.latest.Load() != nil {
		record("old state")
	}
	r.latest.Store(&snapshot{})
	if failures.Add(1) == 1 {
		panic("first run")
	}
	<-ctx.Done()
}

// ping is a process that needs the PID of a pong.
type ping struct{ Process[pingArgs] }

// pingArgs is the argument of a ping.
type pingArgs struct {
	im.Shared
	Pong PID[pong]
}

// Run calls the pong and waits until ctx ends.
func (p *ping) Run(ctx context.Context) {
	record("ping sees " + p.Args().Pong.Proc().Name())
	<-ctx.Done()
}

// Name returns the name of the ping.
func (*ping) Name() string { return "ping" }

// pong is a process that needs the PID of a ping.
type pong struct{ Process[pongArgs] }

// pongArgs is the argument of a pong.
type pongArgs struct {
	im.Shared
	Ping PID[ping]
}

// Run calls the ping and waits until ctx ends.
func (p *pong) Run(ctx context.Context) {
	record("pong sees " + p.Args().Ping.Proc().Name())
	<-ctx.Done()
}

// Name returns the name of the pong.
func (*pong) Name() string { return "pong" }

// slowFailing is a process that panics after 6 minutes on each run.
type slowFailing struct{ Process[None] }

// Run waits 6 minutes, records the time, and panics.
func (*slowFailing) Run(ctx context.Context) {
	select {
	case <-time.After(6 * time.Minute):
	case <-ctx.Done():
		return
	}
	record(time.Now().UTC().Format(time.TimeOnly))
	panic("slow failure")
}

var selfRoot *Supervisor

// stopper is a process that stops its own supervisor.
type stopper struct{ Process[None] }

// Run stops the supervisor and waits until its own context ends.
func (*stopper) Run(ctx context.Context) {
	selfRoot.Stop()
	<-ctx.Done()
	record("stopped")
}

// expectEvents fails t if the recorded events differ from want.
func expectEvents(t *testing.T, want ...string) {
	t.Helper()
	if got := takeEvents(); !slices.Equal(got, want) {
		t.Fatalf("events = %q, want %q", got, want)
	}
}

// expectStarted fails t if the recorded events differ from want in any order.
func expectStarted(t *testing.T, want ...string) {
	t.Helper()
	got := takeEvents()
	slices.Sort(got)
	slices.Sort(want)
	if !slices.Equal(got, want) {
		t.Fatalf("events = %q, want %q in any order", got, want)
	}
}

// TestStart verifies a new value, a stable Proc, and calls from many goroutines.
func TestStart(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		root := NewRoot()
		pid := Start[counter](root, None{})
		process := pid.Proc()
		if pid.Proc() != process || process.count != 0 {
			t.Fatal("Proc did not return one new process")
		}
		var callers sync.WaitGroup
		for range 4 {
			callers.Go(func() {
				for range 100 {
					pid.Proc().Add()
				}
			})
		}
		callers.Wait()
		if got := pid.Proc().Add(); got != 401 {
			t.Fatalf("count = %d, want 401", got)
		}
		<-root.Stop()
	})
}

// TestStopOrder verifies reverse start order for processes and nested supervisors.
func TestStopOrder(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		root := NewRoot()
		Start[first](root, None{})
		nested := StartSupervisor(root)
		Start[second](nested, None{})
		Start[third](root, None{})
		synctest.Wait()
		expectStarted(t, "start first", "start second", "start third")
		<-root.Stop()
		expectEvents(t, "stop third", "stop second", "stop first")
		<-root.Stop()
	})
}

// TestStopNested verifies that a stopped nested supervisor leaves its parent running.
func TestStopNested(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		root := NewRoot()
		Start[first](root, None{})
		nested := StartSupervisor(root)
		Start[second](nested, None{})
		synctest.Wait()
		expectStarted(t, "start first", "start second")
		<-nested.Stop()
		expectEvents(t, "stop second")
		Start[third](nested, None{})
		<-root.Stop()
		expectEvents(t, "stop first")
	})
}

// TestSelfStop verifies that a process can stop its own supervisor.
func TestSelfStop(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		selfRoot = NewRoot()
		Start[stopper](selfRoot, None{})
		synctest.Wait()
		expectEvents(t, "stopped")
		<-selfRoot.Stop()
	})
}

// TestRestartAndEscalate verifies backoff and the root escalation with the last panic value.
func TestRestartAndEscalate(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		failures.Store(0)
		escalated := make(chan any, 1)
		Start[failing](root(func(value any) { escalated <- value }), None{})
		if value := <-escalated; value != int64(6) {
			t.Fatalf("escalated %v, want 6", value)
		}
		expectEvents(t, "00:00:00", "00:00:01", "00:00:03", "00:00:07", "00:00:15", "00:00:31")
	})
}

// TestRestartWindow verifies that restarts older than 5 minutes do not count.
func TestRestartWindow(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		root := root(func(value any) { t.Errorf("escalated %v", value) })
		Start[slowFailing](root, None{})
		time.Sleep(time.Hour)
		<-root.Stop()
		expectEvents(t, "00:06:00", "00:12:01", "00:18:02", "00:24:03", "00:30:04", "00:36:05",
			"00:42:06", "00:48:07", "00:54:08")
	})
}

// TestEscalateToParent verifies that a nested supervisor stops its children and restarts its subtree.
func TestEscalateToParent(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		root := root(func(value any) { t.Errorf("escalated %v", value) })
		nested := StartSupervisor(root)
		Start[first](nested, None{})
		Start[second](nested, None{})
		failures.Store(0)
		Start[failing](nested, None{})
		synctest.Wait()
		expectStarted(t, "start first", "start second", "00:00:00")
		time.Sleep(31 * time.Second)
		synctest.Wait()
		expectEvents(t, "00:00:01", "00:00:03", "00:00:07", "00:00:15", "00:00:31", "stop second", "stop first")
		time.Sleep(time.Second)
		synctest.Wait()
		expectStarted(t, "start first", "start second", "00:00:32")
		<-root.Stop()
		takeEvents()
	})
}

// TestRestartCreatesNewValue verifies a new value with the same arguments and an empty Owner after a restart.
func TestRestartCreatesNewValue(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		failures.Store(0)
		root := NewRoot()
		pid := Start[restarted](root, label{Name: "cache"})
		before := pid.Proc()
		time.Sleep(time.Second)
		synctest.Wait()
		if pid.Proc() == before || before.runs != 1 || pid.Proc().runs != 1 {
			t.Fatal("restart did not store a new value in the same PID")
		}
		expectEvents(t, "cache", "cache")
		<-root.Stop()
	})
}

// TestReserve verifies Proc before start, a second start, and a start cycle.
func TestReserve(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		root := NewRoot()
		pongPID := Reserve[pong]()
		expectPanic(t, "proc: PID not started", func() { pongPID.Proc() })
		pingPID := Start[ping](root, pingArgs{Pong: pongPID})
		StartReserved(root, pongPID, pongArgs{Ping: pingPID})
		expectPanic(t, "proc: PID already started", func() { StartReserved(root, pongPID, pongArgs{Ping: pingPID}) })
		time.Sleep(time.Second)
		synctest.Wait()
		expectStarted(t, "ping sees pong", "pong sees ping")
		<-root.Stop()
	})
}

// expectPanic fails t if call does not panic with want.
func expectPanic(t *testing.T, want string, call func()) {
	t.Helper()
	defer func() {
		if got := recover(); got != want {
			t.Fatalf("panic = %v, want %q", got, want)
		}
	}()
	call()
}

// TestStartAfterStop verifies that a stopped supervisor does not run new processes.
func TestStartAfterStop(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		root := NewRoot()
		<-root.Stop()
		if Start[first](root, None{}).Proc() == nil {
			t.Fatal("Start returned an empty PID")
		}
		synctest.Wait()
		expectEvents(t)
	})
}

// BenchmarkPIDProc measures one PID.Proc call.
func BenchmarkPIDProc(b *testing.B) {
	root := NewRoot()
	pid := Start[counter](root, None{})
	b.ReportAllocs()
	for b.Loop() {
		_ = pid.Proc()
	}
	<-root.Stop()
}
