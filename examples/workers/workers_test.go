package workers

import (
	"testing"
	"time"

	"github.com/lucasavila00/immutago/pkg/im"
	"github.com/lucasavila00/immutago/pkg/supervise"
	"github.com/lucasavila00/immutago/pkg/supervise/supervisetest"
)

// TestPoolJobsAndShutdown checks actual worker output, distribution, and child cancellation.
func TestPoolJobsAndShutdown(t *testing.T) {
	root := supervisetest.NewRoot(t)
	pool, err := supervise.Start[Pool](root, PoolArgs{Workers: 2, Version: 1})
	if err != nil {
		t.Fatal(err)
	}
	await(t, func() bool { return pool.Current().Ready() })
	for i := range 4 {
		result := pool.Current().Process("Hello World")
		if !result.Available || !result.Text.EqualString("HELLO WORLD") || result.Version != 1 || result.Worker != i%2 {
			t.Fatalf("job %d: available=%t text=%q version=%d worker=%d",
				i, result.Available, result.Text.Unsafe(), result.Version, result.Worker)
		}
	}
	stop(t, root)
	result := pool.Current().Process("after shutdown")
	if result.Available || pool.Current().Ready() {
		t.Fatal("stopped workers still accept jobs")
	}
}

// TestStoppedParent checks services whose Init runs but whose Run never starts.
func TestStoppedParent(t *testing.T) {
	root := supervisetest.NewRoot(t)
	stop(t, root)
	worker, err := supervise.Start[Worker](root, WorkerArgs{ID: 0, Version: 1})
	if err != nil {
		t.Fatal(err)
	}
	result := worker.Current().Process("never run")
	if worker.Current().Ready() || result.Available {
		t.Fatal("worker under a stopped group is available")
	}
	pool, err := supervise.Start[Pool](root, PoolArgs{Workers: 2, Version: 1})
	if err != nil {
		t.Fatal(err)
	}
	result = pool.Current().Process("never run")
	if pool.Current().Ready() || result.Available {
		t.Fatal("pool under a stopped group is available")
	}
}

// TestPoolEarlyStop checks shutdown while child loops may still be starting.
func TestPoolEarlyStop(t *testing.T) {
	for range 10 {
		root := supervisetest.NewRoot(t)
		pool, err := supervise.Start[Pool](root, PoolArgs{Workers: 8, Version: 1})
		if err != nil {
			t.Fatal(err)
		}
		stop(t, root)
		result := pool.Current().Process("after early stop")
		if result.Available {
			t.Fatal("early stop left a worker available")
		}
	}
}

// TestDispatcherHandover checks old-version service, readiness, replacement, and failed setup.
func TestDispatcherHandover(t *testing.T) {
	root := supervisetest.NewRoot(t)
	_, dispatcher := startDispatcher(t, root, time.Hour)
	await(t, func() bool {
		dispatcher.Current().Refresh()
		return dispatcher.Current().Version("blue") == 1
	})
	if !dispatcher.Current().Load("blue", 2) {
		t.Fatal("version 2 setup failed")
	}
	old := dispatcher.Current().Process("blue", "Hello")
	if !old.Available || old.Version != 1 || !old.Text.EqualString("HELLO") {
		t.Fatal("staging a replacement changed the serving version")
	}
	await(t, func() bool {
		dispatcher.Current().Refresh()
		return dispatcher.Current().Version("blue") == 2
	})
	next := dispatcher.Current().Process("blue", "Hello")
	if !next.Available || next.Version != 2 || !next.Text.EqualString("hello") {
		t.Fatal("ready replacement did not serve version 2")
	}
	if dispatcher.Current().Load("blue", 3) {
		t.Fatal("unsupported version started")
	}
	unchanged := dispatcher.Current().Process("blue", "Still Version Two")
	if !unchanged.Text.EqualString("still version two") || unchanged.Version != 2 {
		t.Fatal("failed setup replaced the serving version")
	}
	stop(t, root)
	if dispatcher.Current().Load("blue", 1) {
		t.Fatal("stopped dispatcher accepted another version")
	}
}

// TestDispatcherReconfiguration checks background removal while jobs and Load run concurrently.
func TestDispatcherReconfiguration(t *testing.T) {
	root := supervisetest.NewRoot(t)
	config, dispatcher := startDispatcher(t, root, time.Millisecond)
	await(t, func() bool {
		return dispatcher.Current().Version("blue") == 1 && dispatcher.Current().Version("green") == 1
	})
	jobs := make(chan bool, 1)
	go func() {
		valid := true
		for range 100 {
			result := dispatcher.Current().Process("green", "Keep Serving")
			valid = valid && result.Available && result.Text.EqualString("KEEP SERVING")
		}
		jobs <- valid
	}()
	loaded := make(chan bool, 1)
	go func() { loaded <- dispatcher.Current().Load("blue", 2) }()
	names := im.FreezeSlice([]string{"green"})
	config.Current().Set(names)
	await(t, func() bool { return dispatcher.Current().Version("blue") == 0 })
	<-loaded
	if !<-jobs {
		t.Fatal("changing blue interrupted green's jobs")
	}
	dispatcher.Current().Refresh()
	removed := dispatcher.Current().Process("blue", "removed")
	if removed.Available || dispatcher.Current().Load("blue", 1) {
		t.Fatal("removed variant was restored by a concurrent Load")
	}
}

// startDispatcher supplies two configured variants to a new dispatcher.
func startDispatcher(t *testing.T, root *supervise.Group, every time.Duration) (
	supervise.Ref[VariantConfig], supervise.Ref[Dispatcher],
) {
	t.Helper()
	config, err := supervise.Start[VariantConfig](root, supervise.NoArgs{})
	if err != nil {
		t.Fatal(err)
	}
	names := im.FreezeSlice([]string{"blue", "green"})
	config.Current().Set(names)
	dispatcher, err := supervise.Start[Dispatcher](root, DispatcherArgs{
		Config: config, Workers: 2, Every: every,
	})
	if err != nil {
		t.Fatal(err)
	}
	return config, dispatcher
}

// await waits for an observable service state and reports a stuck transition.
func await(t *testing.T, ready func() bool) {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for !ready() {
		if time.Now().After(deadline) {
			t.Fatal("service state did not change in 5 s")
		}
		time.Sleep(time.Millisecond)
	}
}

// stop waits for a service tree and detects shutdown deadlocks.
func stop(t *testing.T, root *supervise.Group) {
	t.Helper()
	select {
	case <-root.Stop():
	case <-time.After(5 * time.Second):
		t.Fatal("service tree did not stop in 5 s")
	}
}
