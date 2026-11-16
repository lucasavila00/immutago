package supervise

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"testing/synctest"
)

// heldCall is a service with a direct call that can outlive its Run.
type heldCall struct {
	Service[NoArgs]
	mu    sync.Mutex
	count int
}

var (
	callEntered chan struct{}
	callRelease chan struct{}
	panicRun    chan struct{}
	callRuns    atomic.Int32
)

// Run waits for the test to release its failure.
func (*heldCall) Run(context.Context) {
	callRuns.Add(1)
	<-panicRun
	panic("failure")
}

// Add waits across a failure, then changes the instance on which it was called.
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

// TestCallAfterFailure checks that shutdown does not wait for direct method calls.
func TestCallAfterFailure(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		callEntered, callRelease, panicRun = make(chan struct{}), make(chan struct{}), make(chan struct{})
		callRuns.Store(0)
		root := root(func(any) {}, nil)
		ref, _ := Start[heldCall](root, NoArgs{})
		before := ref.Borrow()
		result := make(chan int, 1)
		go func() { result <- ref.Borrow().Add() }()
		<-callEntered
		close(panicRun)
		synctest.Wait()
		<-root.Stop()
		if ref.Borrow() != before || callRuns.Load() != 1 {
			t.Fatal("failure changed the service value or ran it again")
		}
		close(callRelease)
		if <-result != 1 || before.Count() != 1 || ref.Borrow().Count() != 1 {
			t.Fatal("old call did not stay on its original instance")
		}
		<-root.Stop()
	})
}

// TestRefCallAllocations checks the direct call path after the service has started.
func TestRefCallAllocations(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		root := NewRoot(nil)
		ref, _ := Start[counter](root, NoArgs{})
		synctest.Wait()
		if allocations := testing.AllocsPerRun(100, func() { ref.Borrow().Add() }); allocations != 0 {
			t.Fatalf("direct call allocations = %v, want 0", allocations)
		}
		<-root.Stop()
	})
}
