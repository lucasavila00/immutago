package supervise

import (
	"context"
	"errors"
	"runtime"
	"sync/atomic"
	"testing"
	"testing/synctest"
	"time"
)

// initHooks controls one service's initialization and run in the runtime tests.
type initHooks struct {
	init func(*initializing) error
	run  func(*initializing, context.Context)
}

// initializing records the state that Init sets before the runtime publishes it.
type initializing struct {
	Service[ConcurrentSafe[*initHooks]]
	state int
}

// Init calls the setup selected by the test.
func (p *initializing) Init() error { return p.Args().Value().init(p) }

// Run calls the test's run function, or waits until the service stops.
func (p *initializing) Run(ctx context.Context) {
	if run := p.Args().Value().run; run != nil {
		run(p, ctx)
		return
	}
	<-ctx.Done()
}

// expectInitEnded checks cleanup of an initialized candidate and its staged group.
func expectInitEnded(t *testing.T, p *initializing) {
	t.Helper()
	p.mu.Lock()
	ended, children := p.ended, p.children
	p.mu.Unlock()
	if !ended {
		t.Error("candidate did not end")
	}
	if children != nil {
		select {
		case <-children.stopped:
		default:
			t.Error("candidate keeps a staged group")
		}
	}
}

// TestInitBeforePublication keeps Borrow and Run unavailable until Init succeeds.
func TestInitBeforePublication(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		root := NewRoot(nil)
		ref := Reserve[initializing]()
		entered, release := make(chan struct{}), make(chan struct{})
		ran := make(chan int, 1)
		result := make(chan error, 1)
		hooks := &initHooks{
			init: func(p *initializing) error {
				close(entered)
				<-release
				p.state = 17
				return nil
			},
			run: func(p *initializing, ctx context.Context) { ran <- p.state; <-ctx.Done() },
		}
		go func() { result <- StartReserved(root, ref, AssumeConcurrentSafe(hooks)) }()
		<-entered
		expectPanic(t, "supervise: Ref not started", func() { ref.Borrow() })
		select {
		case <-ran:
			t.Fatal("Run started before Init returned")
		default:
		}
		close(release)
		if err := <-result; err != nil {
			t.Fatal(err)
		}
		if ref.Borrow().state != 17 || <-ran != 17 {
			t.Fatal("Init state was not published to Borrow and Run")
		}
		<-root.Stop()
	})
}

// TestInitErrorAndRetry checks exact errors, a zero Start result, and reuse of a reserved Ref.
func TestInitErrorAndRetry(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		root := NewRoot(nil)
		failure := errors.New("initial setup")
		hooks := &initHooks{init: func(*initializing) error { return failure }}
		if ref, err := Start[initializing](root, AssumeConcurrentSafe(hooks)); err != failure || !ref.IsZero() {
			t.Fatalf("Start = %v, %v; want zero Ref and original error", ref, err)
		}
		ref := Reserve[initializing]()
		if err := StartReserved(root, ref, AssumeConcurrentSafe(hooks)); err != failure {
			t.Fatalf("StartReserved error = %v", err)
		}
		if ref.IsZero() {
			t.Fatal("reserved Ref is zero after an Init error")
		}
		expectPanic(t, "supervise: Ref not started", func() { ref.Borrow() })
		hooks.init = func(p *initializing) error { p.state = 3; return nil }
		if err := StartReserved(root, ref, AssumeConcurrentSafe(hooks)); err != nil {
			t.Fatal(err)
		}
		if ref.Borrow().state != 3 {
			t.Fatal("retry did not publish its initialized value")
		}
		<-root.Stop()
	})
}

// TestInitFailureCleansChildren checks errors, panics, and Goexit before a candidate is published.
func TestInitFailureCleansChildren(t *testing.T) {
	for _, mode := range []string{"error", "panic", "Goexit"} {
		t.Run(mode, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) { checkInitFailureChildren(t, mode) })
		})
	}
}

// checkInitFailureChildren stages a child and a nested group, then checks cleanup for one Init failure.
func checkInitFailureChildren(t *testing.T, mode string) {
	root := NewRoot(nil)
	failure := errors.New("initial setup")
	var candidates []*initializing
	var nested *Group
	var runs atomic.Int32
	child := &initHooks{
		init: func(p *initializing) error {
			candidates = append(candidates, p)
			p.Children()
			return nil
		},
		run: func(*initializing, context.Context) { runs.Add(1) },
	}
	hooks := &initHooks{init: func(p *initializing) error {
		candidates = append(candidates, p)
		if _, err := Start[initializing](p.Children(), AssumeConcurrentSafe(child)); err != nil {
			return err
		}
		nested = StartGroup(p.Children())
		if _, err := Start[initializing](nested, AssumeConcurrentSafe(child)); err != nil {
			return err
		}
		switch mode {
		case "panic":
			panic(failure)
		case "Goexit":
			runtime.Goexit()
		}
		return failure
	}}
	ref := Reserve[initializing]()
	var err error
	var panicValue any
	finished := make(chan struct{})
	go func() {
		defer close(finished)
		defer func() { panicValue = recover() }()
		err = StartReserved(root, ref, AssumeConcurrentSafe(hooks))
	}()
	<-finished
	if mode == "error" && err != failure || mode == "panic" && panicValue != failure {
		t.Fatalf("error = %v, panic = %v", err, panicValue)
	}
	if mode == "Goexit" && (err != nil || panicValue != nil) {
		t.Fatalf("Goexit became an error or panic: %v, %v", err, panicValue)
	}
	expectPanic(t, "supervise: Ref not started", func() { ref.Borrow() })
	for _, candidate := range candidates {
		expectInitEnded(t, candidate)
	}
	if runs.Load() != 0 {
		t.Fatal("a staged child ran after failed Init")
	}
	retained := len(nested.children)
	late, lateErr := Start[initializing](nested, AssumeConcurrentSafe(child))
	if lateErr != nil {
		t.Fatal(lateErr)
	}
	expectInitEnded(t, late.Borrow())
	if len(nested.children) != retained {
		t.Error("failed Init left a nested group that retains late children")
	}
	<-nested.Stop()
	<-root.Stop()
}

// TestInitDuplicateStart rejects an existing Ref before it calls Init again.
func TestInitDuplicateStart(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		parent := &Group{stopped: make(chan struct{})}
		hooks := &initHooks{init: func(*initializing) error { return nil }}
		ref, err := Start[initializing](parent, AssumeConcurrentSafe(hooks))
		if err != nil {
			t.Fatal(err)
		}
		hooks.init = func(*initializing) error { t.Error("duplicate start called Init"); return errors.New("duplicate") }
		expectPanic(t, "supervise: Ref already started", func() {
			if err := StartReserved(parent, ref, AssumeConcurrentSafe(hooks)); err != nil {
				panic(err)
			}
		})
		<-parent.Stop()
	})
}

// TestInitConcurrentStarts cleans the candidate that loses publication without changing the winner.
func TestInitConcurrentStarts(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		parent := &Group{stopped: make(chan struct{})}
		ref := Reserve[initializing]()
		entered := make(chan *initializing, 2)
		release := make(chan struct{})
		results := make(chan any, 2)
		hooks := &initHooks{init: func(p *initializing) error {
			p.Children()
			entered <- p
			<-release
			return nil
		}}
		for range 2 {
			go func() {
				defer func() { results <- recover() }()
				if err := StartReserved(parent, ref, AssumeConcurrentSafe(hooks)); err != nil {
					panic(err)
				}
			}()
		}
		first, second := <-entered, <-entered
		close(release)
		a, b := <-results, <-results
		if !(a == nil && b == "supervise: Ref already started" || b == nil && a == "supervise: Ref already started") {
			t.Fatalf("concurrent results = %v, %v", a, b)
		}
		loser := first
		if ref.Borrow() == first {
			loser = second
		}
		expectInitEnded(t, loser)
		if len(parent.children) != 1 {
			t.Fatalf("parent has %d entries, want 1", len(parent.children))
		}
		<-parent.Stop()
		expectInitEnded(t, ref.Borrow())
	})
}

// TestInitRestartFailures keeps the previous value until a later initialization succeeds.
func TestInitRestartFailures(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		reports := make(chan any, 3)
		root := NewRoot(func(value any, _ []byte) { reports <- value })
		failure := errors.New("restart setup")
		panicValue := errors.New("restart panic")
		var attempts atomic.Int32
		hooks := &initHooks{
			init: func(p *initializing) error {
				p.state = int(attempts.Add(1))
				switch p.state {
				case 2:
					return failure
				case 3:
					panic(panicValue)
				}
				return nil
			},
			run: func(p *initializing, ctx context.Context) {
				if p.state == 1 {
					panic("first run")
				}
				<-ctx.Done()
			},
		}
		ref, err := Start[initializing](root, AssumeConcurrentSafe(hooks))
		if err != nil {
			t.Fatal(err)
		}
		before := ref.Borrow()
		if value := <-reports; value != "first run" {
			t.Fatalf("first failure = %v", value)
		}
		time.Sleep(time.Second)
		if value := <-reports; value != failure || ref.Borrow() != before {
			t.Fatalf("Init error = %v; Ref changed = %t", value, ref.Borrow() != before)
		}
		time.Sleep(2 * time.Second)
		if value := <-reports; value != panicValue || ref.Borrow() != before {
			t.Fatalf("Init panic = %v; Ref changed = %t", value, ref.Borrow() != before)
		}
		time.Sleep(4 * time.Second)
		synctest.Wait()
		if ref.Borrow() == before || ref.Borrow().state != 4 || attempts.Load() != 4 {
			t.Fatal("successful restart did not publish its initialized value")
		}
		<-root.Stop()
	})
}

// TestInitRestartErrorsEscalate counts initialization errors in the existing restart limit.
func TestInitRestartErrorsEscalate(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		escalated := make(chan any, 1)
		parent := root(func(value any) { escalated <- value }, nil)
		failure := errors.New("restart setup")
		var attempts atomic.Int32
		hooks := &initHooks{
			init: func(*initializing) error {
				if attempts.Add(1) > 1 {
					return failure
				}
				return nil
			},
			run: func(*initializing, context.Context) { panic("first run") },
		}
		ref, err := Start[initializing](parent, AssumeConcurrentSafe(hooks))
		if err != nil {
			t.Fatal(err)
		}
		before := ref.Borrow()
		time.Sleep(31 * time.Second)
		if value := <-escalated; value != failure || attempts.Load() != 6 || ref.Borrow() != before {
			t.Fatalf("escalation = %v, attempts = %d", value, attempts.Load())
		}
		<-parent.Stop()
	})
}

// TestInitRestartGoexit cleans the unpublished candidate and lets the watcher finish.
func TestInitRestartGoexit(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		root := NewRoot(nil)
		candidates := make(chan *initializing, 1)
		var attempts atomic.Int32
		hooks := &initHooks{
			init: func(p *initializing) error {
				if attempts.Add(1) > 1 {
					p.Children()
					candidates <- p
					runtime.Goexit()
				}
				return nil
			},
			run: func(*initializing, context.Context) { panic("restart") },
		}
		ref, err := Start[initializing](root, AssumeConcurrentSafe(hooks))
		if err != nil {
			t.Fatal(err)
		}
		before := ref.Borrow()
		time.Sleep(time.Second)
		candidate := <-candidates
		synctest.Wait()
		expectInitEnded(t, candidate)
		if ref.Borrow() != before {
			t.Fatal("Goexit published its candidate")
		}
		<-root.Stop()
	})
}

// TestInitStopBeforePublication cleans a candidate that finishes Init after its parent stopped.
func TestInitStopBeforePublication(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		root := NewRoot(nil)
		entered := make(chan *initializing, 1)
		release := make(chan struct{})
		result := make(chan error, 1)
		var runs atomic.Int32
		hooks := &initHooks{
			init: func(p *initializing) error {
				p.Children()
				entered <- p
				<-release
				return nil
			},
			run: func(*initializing, context.Context) { runs.Add(1) },
		}
		ref := Reserve[initializing]()
		go func() { result <- StartReserved(root, ref, AssumeConcurrentSafe(hooks)) }()
		candidate := <-entered
		<-root.Stop()
		expectPanic(t, "supervise: Ref not started", func() { ref.Borrow() })
		close(release)
		if err := <-result; err != nil {
			t.Fatal(err)
		}
		if ref.Borrow() != candidate || runs.Load() != 0 {
			t.Fatal("first Init did not keep the stopped-parent start contract")
		}
		expectInitEnded(t, candidate)
	})
}

// TestInitStopDuringRestart waits for Init, then cleans its candidate without publication or Run.
func TestInitStopDuringRestart(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		root := NewRoot(nil)
		entered := make(chan *initializing, 1)
		release := make(chan struct{})
		var attempts, runs atomic.Int32
		hooks := &initHooks{
			init: func(p *initializing) error {
				if attempts.Add(1) == 1 {
					return nil
				}
				p.Children()
				entered <- p
				<-release
				return nil
			},
			run: func(*initializing, context.Context) { runs.Add(1); panic("restart") },
		}
		ref, err := Start[initializing](root, AssumeConcurrentSafe(hooks))
		if err != nil {
			t.Fatal(err)
		}
		before := ref.Borrow()
		time.Sleep(time.Second)
		candidate := <-entered
		stopped := root.Stop()
		synctest.Wait()
		select {
		case <-stopped:
			t.Error("Stop returned before restart Init finished")
		default:
		}
		close(release)
		<-stopped
		if ref.Borrow() != before || runs.Load() != 1 {
			t.Fatal("stopped restart published or ran its candidate")
		}
		expectInitEnded(t, candidate)
	})
}

// TestInitPendingRelaunch initializes a new value if a pending first value was already stopped.
func TestInitPendingRelaunch(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		parent := &Group{stopped: make(chan struct{})}
		var attempts atomic.Int32
		hooks := &initHooks{init: func(p *initializing) error { p.state = int(attempts.Add(1)); return nil }}
		ref, err := Start[initializing](parent, AssumeConcurrentSafe(hooks))
		if err != nil {
			t.Fatal(err)
		}
		before := ref.Borrow()
		parent.halt()
		expectInitEnded(t, before)
		go parent.serve(context.Background())
		synctest.Wait()
		if ref.Borrow() == before || ref.Borrow().state != 2 {
			t.Fatal("parent relaunched an ended candidate")
		}
		<-parent.Stop()
	})
}

// TestInitStopsStagedGroup checks explicit Stop of a nested group before its parent runs.
func TestInitStopsStagedGroup(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		root := NewRoot(nil)
		var child *initializing
		hooks := &initHooks{init: func(p *initializing) error {
			nested := StartGroup(p.Children())
			_, err := Start[initializing](nested, AssumeConcurrentSafe(&initHooks{init: func(p *initializing) error {
				child = p
				p.Children()
				return nil
			}}))
			<-nested.Stop()
			return err
		}}
		if _, err := Start[initializing](root, AssumeConcurrentSafe(hooks)); err != nil {
			t.Fatal(err)
		}
		expectInitEnded(t, child)
		<-root.Stop()
	})
}

// TestInitUnderStoppedGroups cleans late candidates in groups created before or after parent Stop.
func TestInitUnderStoppedGroups(t *testing.T) {
	for _, after := range []bool{false, true} {
		synctest.Test(t, func(t *testing.T) {
			parent := NewRoot(nil)
			var group *Group
			if !after {
				group = StartGroup(parent)
			}
			<-parent.Stop()
			if after {
				group = StartGroup(parent)
			}
			hooks := &initHooks{init: func(p *initializing) error { p.Children(); return nil }}
			ref, err := Start[initializing](group, AssumeConcurrentSafe(hooks))
			if err != nil {
				t.Fatal(err)
			}
			expectInitEnded(t, ref.Borrow())
			if len(group.children) != 0 {
				t.Fatal("group under a stopped parent retains a late candidate")
			}
			<-group.Stop()
		})
	}
}
