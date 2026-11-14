package proc

import (
	"context"
	"sync/atomic"
	"testing"
	"testing/synctest"
	"time"
)

// pool is a process that starts three children under its own supervisor.
type pool struct{ Process[None] }

// Run starts the children, one of them in a nested supervisor, and waits until ctx ends.
func (p *pool) Run(ctx context.Context) {
	record("start pool")
	sup := p.Supervisor()
	Start[first](sup, None{})
	Start[second](StartSupervisor(sup), None{})
	Start[third](sup, None{})
	<-ctx.Done()
	record("stop pool")
}

var (
	poolRuns    atomic.Int32
	poolRelease chan struct{}
)

// restartedPool is a process that starts two children and panics on its first run.
type restartedPool struct{ Process[None] }

// Run starts the children, panics on the first run when released, and otherwise waits until ctx ends.
func (p *restartedPool) Run(ctx context.Context) {
	Start[first](p.Supervisor(), None{})
	Start[second](p.Supervisor(), None{})
	if poolRuns.Add(1) == 1 {
		<-poolRelease
		panic("pool")
	}
	<-ctx.Done()
}

// failingParent is a process with a child that panics on each run.
type failingParent struct{ Process[None] }

// Run starts the failing child and waits until ctx ends.
func (p *failingParent) Run(ctx context.Context) {
	record("start parent")
	Start[failing](p.Supervisor(), None{})
	<-ctx.Done()
	record("stop parent")
}

// TestChildrenStopAfterRun verifies that a stopped process stops its children last started first.
func TestChildrenStopAfterRun(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		root := NewRoot()
		Start[pool](root, None{})
		synctest.Wait()
		expectStarted(t, "start pool", "start first", "start second", "start third")
		<-root.Stop()
		expectEvents(t, "stop pool", "stop third", "stop second", "stop first")
	})
}

// TestChildrenRestartWithRun verifies that a restart stops the children of the old run and the new run starts them.
func TestChildrenRestartWithRun(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		poolRuns.Store(0)
		poolRelease = make(chan struct{})
		root := root(func(value any) { t.Errorf("escalated %v", value) })
		Start[restartedPool](root, None{})
		synctest.Wait()
		expectStarted(t, "start first", "start second")
		close(poolRelease)
		synctest.Wait()
		expectEvents(t, "stop second", "stop first")
		time.Sleep(time.Second)
		synctest.Wait()
		expectStarted(t, "start first", "start second")
		<-root.Stop()
		expectEvents(t, "stop second", "stop first")
	})
}

// TestChildEscalationFailsRun verifies that a child past its restart limit fails the run of its parent.
func TestChildEscalationFailsRun(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		failures.Store(0)
		root := root(func(value any) { t.Errorf("escalated %v", value) })
		Start[failingParent](root, None{})
		time.Sleep(31 * time.Second)
		synctest.Wait()
		expectEvents(t, "start parent", "00:00:00", "00:00:01", "00:00:03", "00:00:07", "00:00:15", "00:00:31",
			"stop parent")
		time.Sleep(time.Second)
		synctest.Wait()
		expectEvents(t, "start parent", "00:00:32")
		<-root.Stop()
		expectEvents(t, "stop parent")
	})
}
