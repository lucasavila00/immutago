package proc

import (
	"context"
	"slices"
	"sync"
	"testing"
	"testing/synctest"
	"time"
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
type first struct{}

// Run records the start and the stop.
func (*first) Run(ctx context.Context) { track(ctx, "first") }

// second is a process that runs until it stops.
type second struct{}

// Run records the start and the stop.
func (*second) Run(ctx context.Context) { track(ctx, "second") }

// third is a process that runs until it stops.
type third struct{}

// Run records the start and the stop.
func (*third) Run(ctx context.Context) { track(ctx, "third") }

// counter is a process with a mutex-protected count.
type counter struct {
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

// failing is a process that panics on each run.
type failing struct {
	runs int
}

// Run records the run time and panics with the run number.
func (f *failing) Run(context.Context) {
	f.runs++
	record(time.Now().UTC().Format(time.TimeOnly))
	panic(f.runs)
}

// slowFailing is a process that panics after 6 minutes on each run.
type slowFailing struct{}

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
type stopper struct{}

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

// TestStartFromZeroValue verifies the zero value, a stable Proc, and calls from many goroutines.
func TestStartFromZeroValue(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		root := NewRoot()
		pid := Start[counter](root)
		process := pid.Proc()
		if pid.Proc() != process || process.count != 0 {
			t.Fatal("Proc did not return one zero process")
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
		Start[first](root)
		nested := StartSupervisor(root)
		Start[second](nested)
		Start[third](root)
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
		Start[first](root)
		nested := StartSupervisor(root)
		Start[second](nested)
		synctest.Wait()
		expectStarted(t, "start first", "start second")
		<-nested.Stop()
		expectEvents(t, "stop second")
		Start[third](nested)
		<-root.Stop()
		expectEvents(t, "stop first")
	})
}

// TestSelfStop verifies that a process can stop its own supervisor.
func TestSelfStop(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		selfRoot = NewRoot()
		Start[stopper](selfRoot)
		synctest.Wait()
		expectEvents(t, "stopped")
		<-selfRoot.Stop()
	})
}

// TestRestartAndEscalate verifies backoff, state across restarts, and the root escalation.
func TestRestartAndEscalate(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		escalated := make(chan any, 1)
		pid := Start[failing](root(func(value any) { escalated <- value }))
		if value := <-escalated; value != 6 || pid.Proc().runs != 6 {
			t.Fatalf("escalated %v after %d runs, want 6 after 6", value, pid.Proc().runs)
		}
		expectEvents(t, "00:00:00", "00:00:01", "00:00:03", "00:00:07", "00:00:15", "00:00:31")
	})
}

// TestRestartWindow verifies that restarts older than 5 minutes do not count.
func TestRestartWindow(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		root := root(func(value any) { t.Errorf("escalated %v", value) })
		Start[slowFailing](root)
		time.Sleep(time.Hour)
		<-root.Stop()
		expectEvents(t, "00:06:00", "00:12:01", "00:18:02", "00:24:03", "00:30:04", "00:36:05",
			"00:42:06", "00:48:07", "00:54:08")
	})
}

// TestEscalateToParent verifies that a nested supervisor stops its children and restarts on the same values.
func TestEscalateToParent(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		root := root(func(value any) { t.Errorf("escalated %v", value) })
		nested := StartSupervisor(root)
		Start[first](nested)
		Start[second](nested)
		pid := Start[failing](nested)
		synctest.Wait()
		expectStarted(t, "start first", "start second", "00:00:00")
		time.Sleep(31 * time.Second)
		synctest.Wait()
		expectEvents(t, "00:00:01", "00:00:03", "00:00:07", "00:00:15", "00:00:31", "stop second", "stop first")
		time.Sleep(time.Second)
		synctest.Wait()
		expectStarted(t, "start first", "start second", "00:00:32")
		if pid.Proc().runs != 7 {
			t.Fatalf("runs = %d, want 7", pid.Proc().runs)
		}
		<-root.Stop()
		takeEvents()
	})
}

// TestStartAfterStop verifies that a stopped supervisor does not run new processes.
func TestStartAfterStop(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		root := NewRoot()
		<-root.Stop()
		if Start[first](root).Proc() == nil {
			t.Fatal("Start returned an empty PID")
		}
		synctest.Wait()
		expectEvents(t)
	})
}

// BenchmarkPIDProc measures one PID.Proc call.
func BenchmarkPIDProc(b *testing.B) {
	root := NewRoot()
	pid := Start[counter](root)
	b.ReportAllocs()
	for b.Loop() {
		_ = pid.Proc()
	}
	<-root.Stop()
}
