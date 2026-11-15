package supervise

import (
	"context"
	"slices"
	"strings"
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

// track records the start and the stop of a service that runs until ctx ends.
func track(ctx context.Context, name string) {
	record("start " + name)
	<-ctx.Done()
	record("stop " + name)
}

// first is a service that runs until it stops.
type first struct{ Service[NoArgs] }

// Run records the start and the stop.
func (*first) Run(ctx context.Context) { track(ctx, "first") }

// second is a service that runs until it stops.
type second struct{ Service[NoArgs] }

// Run records the start and the stop.
func (*second) Run(ctx context.Context) { track(ctx, "second") }

// third is a service that runs until it stops.
type third struct{ Service[NoArgs] }

// Run records the start and the stop.
func (*third) Run(ctx context.Context) { track(ctx, "third") }

// counter is a service with a mutex-protected count.
type counter struct {
	Service[NoArgs]
	mu    sync.Mutex
	count int
}

// Run waits until the service stops.
func (c *counter) Run(ctx context.Context) { <-ctx.Done() }

// Add increases the count and returns the new count.
func (c *counter) Add() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.count++
	return c.count
}

var failures atomic.Int64

// failing is a service that panics on each run.
type failing struct{ Service[NoArgs] }

// Run records the run time and panics with the number of failures.
func (*failing) Run(context.Context) {
	record(time.Now().UTC().Format(time.TimeOnly))
	panic(failures.Add(1))
}

// label is the argument of a restarted service.
type label struct {
	im.Frozen
	Name string
}

// snapshot is a value that a restarted service publishes.
type snapshot struct{ im.Frozen }

// restarted is a service that panics on its first run.
type restarted struct {
	Service[label]
	runs   int
	latest im.Cell[*snapshot]
}

// Run records the arguments and the state of the value, and panics on the first failure.
func (r *restarted) Run(ctx context.Context) {
	r.runs++
	record(r.Args().Name)
	if r.runs != 1 || r.latest.LoadShared() != nil {
		record("old state")
	}
	r.latest.Store(&snapshot{})
	if failures.Add(1) == 1 {
		panic("first run")
	}
	<-ctx.Done()
}

// slowFailing is a service that panics after 6 minutes on each run.
type slowFailing struct{ Service[NoArgs] }

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

var selfRoot *Group

// stopper is a service that stops its own group.
type stopper struct{ Service[NoArgs] }

// Run stops the group and waits until its own context ends.
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

// TestStart verifies a new value, a stable Borrow, and calls from many goroutines.
func TestStart(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		root := NewRoot(nil)
		ref, err := Start[counter](root, NoArgs{})
		if err != nil {
			t.Fatal(err)
		}
		service := ref.Borrow()
		if ref.Borrow() != service || service.count != 0 {
			t.Fatal("Borrow did not return one new service")
		}
		var callers sync.WaitGroup
		for range 4 {
			callers.Go(func() {
				for range 100 {
					ref.Borrow().Add()
				}
			})
		}
		callers.Wait()
		if got := ref.Borrow().Add(); got != 401 {
			t.Fatalf("count = %d, want 401", got)
		}
		<-root.Stop()
	})
}

// TestStopOrder verifies reverse start order for services and nested groups.
func TestStopOrder(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		root := NewRoot(nil)
		if _, err := Start[first](root, NoArgs{}); err != nil {
			t.Fatal(err)
		}
		nested := StartGroup(root)
		if _, err := Start[second](nested, NoArgs{}); err != nil {
			t.Fatal(err)
		}
		if _, err := Start[third](root, NoArgs{}); err != nil {
			t.Fatal(err)
		}
		synctest.Wait()
		expectStarted(t, "start first", "start second", "start third")
		<-root.Stop()
		expectEvents(t, "stop third", "stop second", "stop first")
		<-root.Stop()
	})
}

// TestStopNested verifies that a stopped nested group leaves its parent running.
func TestStopNested(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		root := NewRoot(nil)
		if _, err := Start[first](root, NoArgs{}); err != nil {
			t.Fatal(err)
		}
		nested := StartGroup(root)
		if _, err := Start[second](nested, NoArgs{}); err != nil {
			t.Fatal(err)
		}
		synctest.Wait()
		expectStarted(t, "start first", "start second")
		<-nested.Stop()
		expectEvents(t, "stop second")
		if _, err := Start[third](nested, NoArgs{}); err != nil {
			t.Fatal(err)
		}
		<-root.Stop()
		expectEvents(t, "stop first")
	})
}

// TestStopNestedReleases verifies that a parent keeps no reference to a nested group after its Stop.
func TestStopNestedReleases(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		root := NewRoot(nil)
		if _, err := Start[first](root, NoArgs{}); err != nil {
			t.Fatal(err)
		}
		for range 3 {
			nested := StartGroup(root)
			if _, err := Start[second](nested, NoArgs{}); err != nil {
				t.Fatal(err)
			}
			synctest.Wait()
			entry := nested.entry
			<-nested.Stop()
			if nested.parent != nil || nested.entry != nil || entry.run != nil {
				t.Error("stopped group keeps its parent links or run callback")
			}
		}
		root.mu.Lock()
		children := root.children.count
		root.mu.Unlock()
		if children != 1 {
			t.Errorf("root keeps %d children, want 1", children)
		}
		takeEvents()
		<-root.Stop()
		expectEvents(t, "stop first")
	})
}

// TestSelfStop verifies that a service can stop its own group.
func TestSelfStop(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		selfRoot = NewRoot(nil)
		if _, err := Start[stopper](selfRoot, NoArgs{}); err != nil {
			t.Fatal(err)
		}
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
		if _, err := Start[failing](root(func(value any) { escalated <- value }, nil), NoArgs{}); err != nil {
			t.Fatal(err)
		}
		if value := <-escalated; value.(*Failure).Value != int64(6) {
			t.Fatalf("escalated %v, want 6", value)
		}
		expectEvents(t, "00:00:00", "00:00:01", "00:00:03", "00:00:07", "00:00:15", "00:00:31")
	})
}

// TestPanicReport verifies that every recovered panic reaches the report with its stack before the escalation.
func TestPanicReport(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		failures.Store(0)
		var mu sync.Mutex
		var reported []any
		escalated := make(chan any, 1)
		report := func(value any, stack []byte) {
			if !strings.Contains(string(stack), "failing") {
				t.Errorf("stack of %v does not name the service:\n%s", value, stack)
			}
			mu.Lock()
			defer mu.Unlock()
			reported = append(reported, value)
		}
		if _, err := Start[failing](root(func(value any) { escalated <- value }, report), NoArgs{}); err != nil {
			t.Fatal(err)
		}
		<-escalated
		takeEvents()
		mu.Lock()
		defer mu.Unlock()
		want := []any{int64(1), int64(2), int64(3), int64(4), int64(5), int64(6)}
		if !slices.Equal(reported, want) {
			t.Fatalf("reported %v, want %v", reported, want)
		}
	})
}

// TestPanicStackCapture checks stack capture only when a reporter needs it.
func TestPanicStackCapture(t *testing.T) {
	run := func(context.Context) { panic("failure") }
	for _, capture := range []bool{false, true} {
		value, stack, failed := attempt(context.Background(), run, capture, nil)
		if !failed || value != "failure" || (len(stack) != 0) != capture {
			t.Fatalf("capture %t: value %v, stack length %d, failed %t", capture, value, len(stack), failed)
		}
	}
}

// TestNestedPanicReport checks that a nested group inherits its reporter.
func TestNestedPanicReport(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		failures.Store(0)
		reported := make(chan any, 1)
		root := NewRoot(func(value any, _ []byte) { reported <- value })
		nested := StartGroup(root)
		if _, err := Start[failing](nested, NoArgs{}); err != nil {
			t.Fatal(err)
		}
		if value := <-reported; value != int64(1) {
			t.Fatalf("reported %v, want 1", value)
		}
		<-root.Stop()
		takeEvents()
	})
}

// TestRestartWindow verifies that restarts older than 5 minutes do not count.
func TestRestartWindow(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		root := root(func(value any) { t.Errorf("escalated %v", value) }, nil)
		if _, err := Start[slowFailing](root, NoArgs{}); err != nil {
			t.Fatal(err)
		}
		time.Sleep(time.Hour)
		<-root.Stop()
		expectEvents(t, "00:06:00", "00:12:01", "00:18:02", "00:24:03", "00:30:04", "00:36:05",
			"00:42:06", "00:48:07", "00:54:08")
	})
}

// TestRestartCreatesNewValue verifies a new value with the same arguments and an empty Owner after a restart.
func TestRestartCreatesNewValue(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		failures.Store(0)
		root := NewRoot(nil)
		ref, _ := Start[restarted](root, label{Name: "cache"})
		before := ref.Borrow()
		synctest.Wait()
		if ref.Borrow() != before || before.runs != 1 {
			t.Fatal("Ref changed before the restart delay ended")
		}
		time.Sleep(time.Second)
		synctest.Wait()
		if ref.Borrow() == before || before.runs != 1 || ref.Borrow().runs != 1 {
			t.Fatal("restart did not store a new value in the same Ref")
		}
		expectEvents(t, "cache", "cache")
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

// TestStartAfterStop verifies that a stopped group does not run new services.
func TestStartAfterStop(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		root := NewRoot(nil)
		<-root.Stop()
		ref, err := Start[first](root, NoArgs{})
		if err != nil {
			t.Fatal(err)
		}
		if ref.Borrow() == nil {
			t.Fatal("Start returned an empty Ref")
		}
		synctest.Wait()
		expectEvents(t)
	})
}

// BenchmarkRefBorrow measures one Ref.Borrow call.
func BenchmarkRefBorrow(b *testing.B) {
	root := NewRoot(nil)
	ref, _ := Start[counter](root, NoArgs{})
	b.ReportAllocs()
	for b.Loop() {
		_ = ref.Borrow()
	}
	<-root.Stop()
}

// TestStopNestedRightAfterStart verifies that a nested group can stop while its parent launches it.
func TestStopNestedRightAfterStart(t *testing.T) {
	root := NewRoot(nil)
	for range 100 {
		<-StartGroup(root).Stop()
	}
	<-root.Stop()
}

// reporterArgs gives a service a trusted function.
type reporterArgs struct {
	Report func(string)
}

// reporter calls its trusted function once.
type reporter struct{ Service[reporterArgs] }

// Run calls the trusted function.
func (r *reporter) Run(context.Context) { r.Args().Report("run") }

// TestTrustedArgument checks that a service receives the trusted value from its arguments.
func TestTrustedArgument(t *testing.T) {
	reported := make(chan string, 1)
	root := NewRoot(nil)
	args := reporterArgs{Report: func(event string) { reported <- event }}
	if _, err := Start[reporter](root, args); err != nil {
		t.Fatal(err)
	}
	if got := <-reported; got != "run" {
		t.Fatalf("reported %q", got)
	}
	<-root.Stop()
}
