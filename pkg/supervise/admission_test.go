package supervise

import (
	"context"
	"slices"
	"sync"
	"sync/atomic"
	"testing"
	"testing/synctest"
)

// stopOnRead requests Stop between a real context read and its use by startup.
type stopOnRead struct {
	context.Context
	group *Group
	when  func() bool
	once  sync.Once
}

// Err lets Stop win an unlocked read gap, or queues it behind startup's lock.
func (c *stopOnRead) Err() error {
	err := c.Context.Err()
	if c.when == nil || c.when() {
		c.once.Do(func() {
			if c.group.mu.TryLock() {
				c.group.stop(nil)
				c.group.mu.Unlock()
			} else {
				go c.group.Stop()
			}
		})
	}
	return err
}

// TestRunAdmissionOrdersStop keeps child activation ordered against Stop.
func TestRunAdmissionOrdersStop(t *testing.T) {
	synctest.Test(t, func(t *testing.T) { checkRunAdmissionOrdersStop(t) })
}

// checkRunAdmissionOrdersStop holds a later sibling while Stop races with Run admission.
func checkRunAdmissionOrdersStop(t *testing.T) {
	t.Helper()
	top := NewRoot(nil)
	parent := newGroup(top, false)
	var inits, runs, childRuns atomic.Int32
	ran := make(chan *initializing, 1)
	ref, err := Start[initializing](parent, &initHooks{
		init: func(p *initializing) error {
			inits.Add(1)
			_, err := Start[initializing](p.UnsafeChildren(), &initHooks{
				init: func(*initializing) error { return nil },
				run: func(_ *initializing, ctx context.Context) {
					<-ctx.Done()
					childRuns.Add(1)
				},
			})
			return err
		},
		run: func(p *initializing, ctx context.Context) {
			runs.Add(1)
			children := p.UnsafeChildren()
			children.mu.Lock()
			active := children.active
			children.mu.Unlock()
			if ctx.Err() != nil || !active {
				t.Error("Run admission did not activate the staged children with a live parent context")
			}
			ran <- p
			<-ctx.Done()
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	entry := parent.children.first
	run := entry.run
	release := make(chan struct{})
	entry.run = func(ctx context.Context) {
		<-release
		run(&stopOnRead{Context: ctx, group: parent, when: func() bool { return inits.Load() == 1 }})
	}
	cleanup, slowRelease := make(chan struct{}), make(chan struct{})
	parent.add(func(ctx context.Context) { <-ctx.Done(); close(cleanup); <-slowRelease }, nil)
	parent.mu.Lock()
	parent.activate()
	parent.mu.Unlock()
	close(release)
	<-cleanup
	value := <-ran
	if ref.Borrow() != value {
		t.Error("Run admission did not publish its initialized value")
	}
	close(slowRelease)
	<-parent.Stop()
	<-top.Stop()
	if inits.Load() != 1 || runs.Load() != 1 || childRuns.Load() != 1 {
		t.Errorf("Init=%d Run=%d child Run=%d; want 1 each", inits.Load(), runs.Load(), childRuns.Load())
	}
}

// TestStagedActivationOrdersStop keeps every activated entry in reverse shutdown order.
func TestStagedActivationOrdersStop(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		top := NewRoot(nil)
		parent := newGroup(top, false)
		lastCleanup, releaseLast := make(chan struct{}), make(chan struct{})
		var mu sync.Mutex
		var order []int
		_, err := Start[initializing](parent, &initHooks{
			init: func(p *initializing) error {
				children := p.UnsafeChildren()
				for index := range 3 {
					_, err := Start[initializing](children, &initHooks{
						init: func(*initializing) error { return nil },
						run: func(_ *initializing, ctx context.Context) {
							<-ctx.Done()
							if index == 2 {
								close(lastCleanup)
								<-releaseLast
							}
							mu.Lock()
							order = append(order, index)
							mu.Unlock()
						},
					})
					if err != nil {
						return err
					}
				}
				children.policy = &stopOnRead{Context: children.policy, group: parent}
				return nil
			},
		})
		if err != nil {
			t.Fatal(err)
		}
		entry := parent.children.first
		run := entry.run
		release := make(chan struct{})
		entry.run = func(ctx context.Context) { <-release; run(ctx) }
		parent.mu.Lock()
		parent.activate()
		parent.mu.Unlock()
		close(release)
		<-lastCleanup
		synctest.Wait()
		mu.Lock()
		if len(order) != 0 {
			t.Errorf("children stopped before the last child's cleanup ended: %v", order)
		}
		mu.Unlock()
		close(releaseLast)
		<-parent.Stop()
		<-top.Stop()
		if !slices.Equal(order, []int{2, 1, 0}) {
			t.Errorf("cleanup order = %v; want [2 1 0]", order)
		}
	})
}
