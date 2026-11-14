package proc

import (
	"context"
	"errors"
	"runtime"
	"sync/atomic"
	"testing"
	"testing/synctest"
	"time"
)

// initHooks controls one process's initialization and run in the runtime tests.
type initHooks struct {
	init func(*initializing) error
	run  func(*initializing, context.Context)
}

// initializing records the state that Init sets before the runtime publishes it.
type initializing struct {
	Process[ConcurrentSafe[*initHooks]]
	state int
}

// Init calls the setup selected by the test.
func (p *initializing) Init() error { return p.Args().Value().init(p) }

// Run calls the test's run function, or waits until the process stops.
func (p *initializing) Run(ctx context.Context) {
	if run := p.Args().Value().run; run != nil {
		run(p, ctx)
		return
	}
	<-ctx.Done()
}

// expectInitEnded checks cleanup of an initialized candidate and its staged supervisor.
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
			t.Error("candidate keeps a staged supervisor")
		}
	}
}

// TestInitBeforePublication keeps Proc and Run unavailable until Init succeeds.
func TestInitBeforePublication(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		root := NewRoot(nil)
		pid := Reserve[initializing]()
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
		go func() { result <- StartReserved(root, pid, AssumeConcurrentSafe(hooks)) }()
		<-entered
		expectPanic(t, "proc: PID not started", func() { pid.Proc() })
		select {
		case <-ran:
			t.Fatal("Run started before Init returned")
		default:
		}
		close(release)
		if err := <-result; err != nil {
			t.Fatal(err)
		}
		if pid.Proc().state != 17 || <-ran != 17 {
			t.Fatal("Init state was not published to Proc and Run")
		}
		<-root.Stop()
	})
}

// TestInitErrorAndRetry checks exact errors, an empty Start result, and reuse of a reserved PID.
func TestInitErrorAndRetry(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		root := NewRoot(nil)
		failure := errors.New("initial setup")
		hooks := &initHooks{init: func(*initializing) error { return failure }}
		if pid, err := Start[initializing](root, AssumeConcurrentSafe(hooks)); err != failure || pid.value != nil {
			t.Fatalf("Start = %v, %v; want empty PID and original error", pid, err)
		}
		pid := Reserve[initializing]()
		if err := StartReserved(root, pid, AssumeConcurrentSafe(hooks)); err != failure {
			t.Fatalf("StartReserved error = %v", err)
		}
		expectPanic(t, "proc: PID not started", func() { pid.Proc() })
		hooks.init = func(p *initializing) error { p.state = 3; return nil }
		if err := StartReserved(root, pid, AssumeConcurrentSafe(hooks)); err != nil {
			t.Fatal(err)
		}
		if pid.Proc().state != 3 {
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
	var nested *Supervisor
	var runs atomic.Int32
	child := &initHooks{
		init: func(p *initializing) error {
			candidates = append(candidates, p)
			p.Supervisor()
			return nil
		},
		run: func(*initializing, context.Context) { runs.Add(1) },
	}
	hooks := &initHooks{init: func(p *initializing) error {
		candidates = append(candidates, p)
		if _, err := Start[initializing](p.Supervisor(), AssumeConcurrentSafe(child)); err != nil {
			return err
		}
		nested = StartSupervisor(p.Supervisor())
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
	pid := Reserve[initializing]()
	var err error
	var panicValue any
	finished := make(chan struct{})
	go func() {
		defer close(finished)
		defer func() { panicValue = recover() }()
		err = StartReserved(root, pid, AssumeConcurrentSafe(hooks))
	}()
	<-finished
	if mode == "error" && err != failure || mode == "panic" && panicValue != failure {
		t.Fatalf("error = %v, panic = %v", err, panicValue)
	}
	if mode == "Goexit" && (err != nil || panicValue != nil) {
		t.Fatalf("Goexit became an error or panic: %v, %v", err, panicValue)
	}
	expectPanic(t, "proc: PID not started", func() { pid.Proc() })
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
	expectInitEnded(t, late.Proc())
	if len(nested.children) != retained {
		t.Error("failed Init left a nested supervisor that retains late children")
	}
	<-nested.Stop()
	<-root.Stop()
}

// TestInitDuplicateStart rejects an existing PID before it calls Init again.
func TestInitDuplicateStart(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		parent := &Supervisor{stopped: make(chan struct{})}
		hooks := &initHooks{init: func(*initializing) error { return nil }}
		pid, err := Start[initializing](parent, AssumeConcurrentSafe(hooks))
		if err != nil {
			t.Fatal(err)
		}
		hooks.init = func(*initializing) error { t.Error("duplicate start called Init"); return errors.New("duplicate") }
		expectPanic(t, "proc: PID already started", func() {
			if err := StartReserved(parent, pid, AssumeConcurrentSafe(hooks)); err != nil {
				panic(err)
			}
		})
		<-parent.Stop()
	})
}

// TestInitConcurrentStarts cleans the candidate that loses publication without changing the winner.
func TestInitConcurrentStarts(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		parent := &Supervisor{stopped: make(chan struct{})}
		pid := Reserve[initializing]()
		entered := make(chan *initializing, 2)
		release := make(chan struct{})
		results := make(chan any, 2)
		hooks := &initHooks{init: func(p *initializing) error {
			p.Supervisor()
			entered <- p
			<-release
			return nil
		}}
		for range 2 {
			go func() {
				defer func() { results <- recover() }()
				if err := StartReserved(parent, pid, AssumeConcurrentSafe(hooks)); err != nil {
					panic(err)
				}
			}()
		}
		first, second := <-entered, <-entered
		close(release)
		a, b := <-results, <-results
		if !(a == nil && b == "proc: PID already started" || b == nil && a == "proc: PID already started") {
			t.Fatalf("concurrent results = %v, %v", a, b)
		}
		loser := first
		if pid.Proc() == first {
			loser = second
		}
		expectInitEnded(t, loser)
		if len(parent.children) != 1 {
			t.Fatalf("parent has %d entries, want 1", len(parent.children))
		}
		<-parent.Stop()
		expectInitEnded(t, pid.Proc())
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
		pid, err := Start[initializing](root, AssumeConcurrentSafe(hooks))
		if err != nil {
			t.Fatal(err)
		}
		before := pid.Proc()
		if value := <-reports; value != "first run" {
			t.Fatalf("first failure = %v", value)
		}
		time.Sleep(time.Second)
		if value := <-reports; value != failure || pid.Proc() != before {
			t.Fatalf("Init error = %v; PID changed = %t", value, pid.Proc() != before)
		}
		time.Sleep(2 * time.Second)
		if value := <-reports; value != panicValue || pid.Proc() != before {
			t.Fatalf("Init panic = %v; PID changed = %t", value, pid.Proc() != before)
		}
		time.Sleep(4 * time.Second)
		synctest.Wait()
		if pid.Proc() == before || pid.Proc().state != 4 || attempts.Load() != 4 {
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
		pid, err := Start[initializing](parent, AssumeConcurrentSafe(hooks))
		if err != nil {
			t.Fatal(err)
		}
		before := pid.Proc()
		time.Sleep(31 * time.Second)
		if value := <-escalated; value != failure || attempts.Load() != 6 || pid.Proc() != before {
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
					p.Supervisor()
					candidates <- p
					runtime.Goexit()
				}
				return nil
			},
			run: func(*initializing, context.Context) { panic("restart") },
		}
		pid, err := Start[initializing](root, AssumeConcurrentSafe(hooks))
		if err != nil {
			t.Fatal(err)
		}
		before := pid.Proc()
		time.Sleep(time.Second)
		candidate := <-candidates
		synctest.Wait()
		expectInitEnded(t, candidate)
		if pid.Proc() != before {
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
				p.Supervisor()
				entered <- p
				<-release
				return nil
			},
			run: func(*initializing, context.Context) { runs.Add(1) },
		}
		pid := Reserve[initializing]()
		go func() { result <- StartReserved(root, pid, AssumeConcurrentSafe(hooks)) }()
		candidate := <-entered
		<-root.Stop()
		expectPanic(t, "proc: PID not started", func() { pid.Proc() })
		close(release)
		if err := <-result; err != nil {
			t.Fatal(err)
		}
		if pid.Proc() != candidate || runs.Load() != 0 {
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
				p.Supervisor()
				entered <- p
				<-release
				return nil
			},
			run: func(*initializing, context.Context) { runs.Add(1); panic("restart") },
		}
		pid, err := Start[initializing](root, AssumeConcurrentSafe(hooks))
		if err != nil {
			t.Fatal(err)
		}
		before := pid.Proc()
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
		if pid.Proc() != before || runs.Load() != 1 {
			t.Fatal("stopped restart published or ran its candidate")
		}
		expectInitEnded(t, candidate)
	})
}

// TestInitPendingRelaunch initializes a new value if a pending first value was already stopped.
func TestInitPendingRelaunch(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		parent := &Supervisor{stopped: make(chan struct{})}
		var attempts atomic.Int32
		hooks := &initHooks{init: func(p *initializing) error { p.state = int(attempts.Add(1)); return nil }}
		pid, err := Start[initializing](parent, AssumeConcurrentSafe(hooks))
		if err != nil {
			t.Fatal(err)
		}
		before := pid.Proc()
		parent.halt()
		expectInitEnded(t, before)
		go parent.serve(context.Background())
		synctest.Wait()
		if pid.Proc() == before || pid.Proc().state != 2 {
			t.Fatal("parent relaunched an ended candidate")
		}
		<-parent.Stop()
	})
}

// TestInitStopsStagedGroup checks explicit Stop of a nested supervisor before its parent runs.
func TestInitStopsStagedGroup(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		root := NewRoot(nil)
		var child *initializing
		hooks := &initHooks{init: func(p *initializing) error {
			nested := StartSupervisor(p.Supervisor())
			_, err := Start[initializing](nested, AssumeConcurrentSafe(&initHooks{init: func(p *initializing) error {
				child = p
				p.Supervisor()
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
			var group *Supervisor
			if !after {
				group = StartSupervisor(parent)
			}
			<-parent.Stop()
			if after {
				group = StartSupervisor(parent)
			}
			hooks := &initHooks{init: func(p *initializing) error { p.Supervisor(); return nil }}
			pid, err := Start[initializing](group, AssumeConcurrentSafe(hooks))
			if err != nil {
				t.Fatal(err)
			}
			expectInitEnded(t, pid.Proc())
			if len(group.children) != 0 {
				t.Fatal("group under a stopped parent retains a late candidate")
			}
			<-group.Stop()
		})
	}
}
