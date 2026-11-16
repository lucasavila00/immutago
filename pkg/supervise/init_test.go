package supervise

import (
	"context"
	"errors"
	"runtime"
	"sync/atomic"
	"testing"
	"testing/synctest"
)

// initHooks controls one service's initialization and run in the runtime tests.
type initHooks struct {
	init func(*initializing) error
	run  func(*initializing, context.Context)
}

// initializing records the state that Init sets before the runtime publishes it.
type initializing struct {
	Service[*initHooks]
	state int
}

// Init calls the setup selected by the test.
func (p *initializing) Init() error { return p.Args().init(p) }

// Run calls the test's run function, or waits until the service stops.
func (p *initializing) Run(ctx context.Context) {
	if run := p.Args().run; run != nil {
		run(p, ctx)
		return
	}
	<-ctx.Done()
}

// expectInitEnded checks cleanup of an initialized candidate and its staged group.
func expectInitEnded(t *testing.T, p *initializing) {
	t.Helper()
	p.owner.mu.Lock()
	ended, children := p.ended, p.children
	p.owner.mu.Unlock()
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

// TestInitBeforePublication keeps Start and Run unavailable until Init succeeds.
func TestInitBeforePublication(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		root := NewRoot(nil)
		entered, release := make(chan struct{}), make(chan struct{})
		ran := make(chan int, 1)
		result := make(chan Ref[initializing], 1)
		hooks := &initHooks{
			init: func(p *initializing) error { close(entered); <-release; p.state = 17; return nil },
			run:  func(p *initializing, ctx context.Context) { ran <- p.state; <-ctx.Done() },
		}
		go func() {
			ref, err := Start[initializing](root, hooks)
			if err != nil {
				panic(err)
			}
			result <- ref
		}()
		<-entered
		select {
		case <-result:
			t.Fatal("Start returned before Init")
		default:
		}
		select {
		case <-ran:
			t.Fatal("Run started before Init")
		default:
		}
		close(release)
		ref := <-result
		if ref.Borrow().state != 17 || <-ran != 17 {
			t.Fatal("Init state was not published")
		}
		<-root.Stop()
	})
}

// TestInitErrorAndNewStart checks first errors and a later independent Start.
func TestInitErrorAndNewStart(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		root := NewRoot(nil)
		failure := errors.New("initial setup")
		hooks := &initHooks{init: func(*initializing) error { return failure }}
		if ref, err := Start[initializing](root, hooks); err != failure || !ref.IsZero() {
			t.Fatalf("Start = %v, %v; want zero Ref and original error", ref, err)
		}
		hooks.init = func(p *initializing) error { p.state = 3; return nil }
		ref, err := Start[initializing](root, hooks)
		if err != nil || ref.Borrow().state != 3 {
			t.Fatalf("Start failed: %v", err)
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
		run: func(_ *initializing, ctx context.Context) {
			if ctx.Err() == nil {
				t.Error("staged child started with an active context after failed Init")
			}
			runs.Add(1)
		},
	}
	hooks := &initHooks{init: func(p *initializing) error {
		candidates = append(candidates, p)
		if _, err := Start[initializing](p.Children(), child); err != nil {
			return err
		}
		nested = StartGroup(p.Children())
		if _, err := Start[initializing](nested, child); err != nil {
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
	var ref Ref[initializing]
	var err error
	var panicValue any
	finished := make(chan struct{})
	go func() {
		defer close(finished)
		defer func() { panicValue = recover() }()
		ref, err = Start[initializing](root, hooks)
	}()
	<-finished
	if mode == "error" && err != failure || mode == "panic" && panicValue != failure {
		t.Fatalf("error = %v, panic = %v", err, panicValue)
	}
	if mode == "Goexit" && (err != nil || panicValue != nil) {
		t.Fatalf("Goexit became an error or panic: %v, %v", err, panicValue)
	}
	if !ref.IsZero() {
		t.Fatal("failed or pending Init published a Ref")
	}
	for _, candidate := range candidates {
		expectInitEnded(t, candidate)
	}
	if runs.Load() != 2 {
		t.Fatal("staged children did not each run cleanup after failed Init")
	}
	retained := nested.children.count
	late, lateErr := Start[initializing](nested, child)
	if lateErr != nil {
		t.Fatal(lateErr)
	}
	expectInitEnded(t, late.Borrow())
	if nested.children.count != retained {
		t.Error("failed Init left a nested group that retains late children")
	}
	<-nested.Stop()
	<-root.Stop()
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
			run: func(_ *initializing, ctx context.Context) {
				if ctx.Err() == nil {
					t.Error("late setup ran with an active context")
				}
				runs.Add(1)
			},
		}
		var ref Ref[initializing]
		go func() {
			var err error
			ref, err = Start[initializing](root, hooks)
			result <- err
		}()
		candidate := <-entered
		<-root.Stop()
		if !ref.IsZero() {
			t.Fatal("failed or pending Init published a Ref")
		}
		close(release)
		if err := <-result; err != nil {
			t.Fatal(err)
		}
		if ref.Borrow() != candidate || runs.Load() != 1 {
			t.Fatal("first Init did not keep the stopped-parent start contract")
		}
		expectInitEnded(t, candidate)
	})
}

// TestInitPendingActivation uses the first initialized value exactly once.
func TestInitPendingActivation(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		top := NewRoot(nil)
		parent := newGroup(top, false)
		var attempts atomic.Int32
		hooks := &initHooks{init: func(p *initializing) error { p.state = int(attempts.Add(1)); return nil }}
		ref, err := Start[initializing](parent, hooks)
		if err != nil {
			t.Fatal(err)
		}
		before := ref.Borrow()
		parent.mu.Lock()
		parent.activate()
		parent.mu.Unlock()
		parent.mu.Lock()
		parent.activate()
		parent.mu.Unlock()
		synctest.Wait()
		if ref.Borrow() != before || attempts.Load() != 1 {
			t.Fatal("activation replaced pending setup")
		}
		<-parent.Stop()
		<-top.Stop()
	})
}

// TestInitStopsStagedGroup checks explicit Stop of a nested group before its parent runs.
func TestInitStopsStagedGroup(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		root := NewRoot(nil)
		var child *initializing
		hooks := &initHooks{init: func(p *initializing) error {
			nested := StartGroup(p.Children())
			_, err := Start[initializing](nested, &initHooks{init: func(p *initializing) error {
				child = p
				p.Children()
				return nil
			}})
			<-nested.Stop()
			return err
		}}
		if _, err := Start[initializing](root, hooks); err != nil {
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
			ref, err := Start[initializing](group, hooks)
			if err != nil {
				t.Fatal(err)
			}
			expectInitEnded(t, ref.Borrow())
			if group.children.count != 0 {
				t.Fatal("group under a stopped parent retains a late candidate")
			}
			<-group.Stop()
		})
	}
}
