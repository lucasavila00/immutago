package proc

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"testing/synctest"
	"time"
)

// heldCall is a process with a direct call that can outlive its Run.
type heldCall struct {
	Process[None]
	mu    sync.Mutex
	count int
}

var (
	callEntered chan struct{}
	callRelease chan struct{}
	panicRun    chan struct{}
	callRuns    atomic.Int32
)

// Run waits for a panic on the first attempt and cancellation after the restart.
func (*heldCall) Run(ctx context.Context) {
	if callRuns.Add(1) == 1 {
		<-panicRun
		panic("restart")
	}
	<-ctx.Done()
}

// Add waits across a restart, then changes the instance on which it was called.
func (p *heldCall) Add() int {
	close(callEntered)
	<-callRelease
	p.mu.Lock()
	defer p.mu.Unlock()
	p.count++
	return p.count
}

// Count reads the state of one instance.
func (p *heldCall) Count() int {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.count
}

// TestCallAcrossRestart checks that an old call can finish without changing the new instance.
func TestCallAcrossRestart(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		callEntered, callRelease, panicRun = make(chan struct{}), make(chan struct{}), make(chan struct{})
		callRuns.Store(0)
		root := NewRoot(nil)
		pid := Start[heldCall](root, None{})
		before := pid.Proc()
		result := make(chan int, 1)
		go func() { result <- pid.Proc().Add() }()
		<-callEntered
		close(panicRun)
		synctest.Wait()
		time.Sleep(time.Second)
		synctest.Wait()
		if pid.Proc() == before {
			t.Fatal("restart did not publish a fresh instance")
		}
		close(callRelease)
		if <-result != 1 || before.Count() != 1 || pid.Proc().Count() != 0 {
			t.Fatal("old call did not stay on its original instance")
		}
		<-root.Stop()
	})
}

// TestPIDCallAllocations checks the direct call path after the process has started.
func TestPIDCallAllocations(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		root := NewRoot(nil)
		pid := Start[counter](root, None{})
		synctest.Wait()
		if allocations := testing.AllocsPerRun(100, func() { pid.Proc().Add() }); allocations != 0 {
			t.Fatalf("direct call allocations = %v, want 0", allocations)
		}
		<-root.Stop()
	})
}
