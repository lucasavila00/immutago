// Package proc runs supervised processes.
//
// A process is a struct that embeds im.Process and has a Run(context.Context)
// method with a pointer receiver. Start creates the process from its zero value,
// runs Run in a new goroutine, and returns a Ref. Callers keep the Ref and call
// exported methods directly on the result of Ref.Get. The immutago analyzer
// proves that these methods take and return only immutable values and Refs.
//
// A supervisor runs its children until Stop. If Run returns, the process ends.
// If Run panics, the supervisor runs it again on the same value after a delay of
// 1 s that doubles for each restart in the last 5 minutes. A sixth panic in 5
// minutes stops all children of the supervisor in reverse start order and
// escalates the panic to the parent supervisor, which restarts the subtree with
// the same rule. At the root supervisor, the escalation panics again with the
// last panic value, and the program exits.
package proc

import (
	"context"
	"slices"
	"sync"
	"sync/atomic"
	"time"
)

const (
	restartLimit  = 5
	restartWindow = 5 * time.Minute
)

// Supervisor runs processes and nested supervisors.
type Supervisor struct {
	mu       sync.Mutex
	children []*child
	ctx      context.Context
	failure  chan any
	stopping bool
	stopped  chan struct{}
}

// child is one process or nested supervisor and its current goroutine.
type child struct {
	run  func(context.Context)
	stop context.CancelFunc
	done chan struct{}
}

// Ref refers to a process that Start created.
type Ref[T any] struct {
	value *atomic.Pointer[T]
}

// Get returns the process with one atomic load.
func (r Ref[T]) Get() *T { return r.value.Load() }

// New returns a root supervisor.
func New() *Supervisor { return root(func(value any) { panic(value) }) }

// root returns a root supervisor that calls exit with an escalated panic value.
func root(exit func(any)) *Supervisor {
	s := &Supervisor{stopped: make(chan struct{})}
	go func() {
		if value, failed := attempt(context.Background(), s.serve); failed {
			exit(value)
		}
	}()
	return s
}

// Supervise returns a new supervisor that s runs as a child.
func (s *Supervisor) Supervise() *Supervisor {
	nested := &Supervisor{stopped: make(chan struct{})}
	s.add(nested.serve)
	return nested
}

// Start creates a T from its zero value and runs its Run method under parent.
// If parent is stopped, Start returns the Ref and does not run the process.
func Start[T any, P interface {
	*T
	Run(context.Context)
}](parent *Supervisor) Ref[T] {
	value := new(T)
	ref := Ref[T]{value: new(atomic.Pointer[T])}
	ref.value.Store(value)
	parent.add(P(value).Run)
	return ref
}

// Stop stops the children of s in reverse start order and does not wait.
// The returned channel closes when all children have stopped.
// A process can stop its own supervisor from Run.
func (s *Supervisor) Stop() <-chan struct{} {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.stopping {
		s.stopping = true
		go func() {
			s.halt()
			close(s.stopped)
		}()
	}
	return s.stopped
}

// add records a child and runs it if s is running.
func (s *Supervisor) add(run func(context.Context)) {
	done := make(chan struct{})
	close(done)
	c := &child{run: run, stop: func() {}, done: done}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.children = append(s.children, c)
	if s.ctx != nil && !s.stopping {
		s.launch(c)
	}
}

// launch runs c in a new goroutine. The caller holds s.mu.
func (s *Supervisor) launch(c *child) {
	ctx, cancel := context.WithCancel(s.ctx)
	c.stop, c.done = cancel, make(chan struct{})
	go watch(ctx, c.run, s.failure, c.done)
}

// serve runs one incarnation of s until its parent cancels ctx, s stops, or a child escalates.
func (s *Supervisor) serve(ctx context.Context) {
	s.mu.Lock()
	if s.stopping {
		s.mu.Unlock()
		return
	}
	failure := make(chan any, 1)
	s.ctx, s.failure = context.WithoutCancel(ctx), failure
	for _, c := range s.children {
		s.launch(c)
	}
	s.mu.Unlock()
	var value any
	failed := false
	select {
	case <-ctx.Done():
	case <-s.stopped:
	case value = <-failure:
		failed = true
	}
	s.mu.Lock()
	s.ctx = nil
	s.mu.Unlock()
	s.halt()
	if failed {
		panic(value)
	}
}

// halt stops the children of s one at a time, last started first.
func (s *Supervisor) halt() {
	s.mu.Lock()
	children := s.children
	s.mu.Unlock()
	for _, c := range slices.Backward(children) {
		s.mu.Lock()
		stop, done := c.stop, c.done
		s.mu.Unlock()
		stop()
		<-done
	}
}

// watch runs run until it returns or ctx ends, and restarts it after a panic.
func watch(ctx context.Context, run func(context.Context), failure chan<- any, done chan<- struct{}) {
	defer close(done)
	var restarts []time.Time
	for {
		value, failed := attempt(ctx, run)
		if !failed || ctx.Err() != nil {
			return
		}
		now := time.Now()
		restarts = slices.DeleteFunc(restarts, func(at time.Time) bool { return now.Sub(at) >= restartWindow })
		if len(restarts) == restartLimit {
			select {
			case failure <- value:
			default:
			}
			return
		}
		delay := time.NewTimer(time.Second << len(restarts))
		restarts = append(restarts, now)
		select {
		case <-delay.C:
		case <-ctx.Done():
			delay.Stop()
			return
		}
	}
}

// attempt calls run and returns the panic value if run panics.
func attempt(ctx context.Context, run func(context.Context)) (value any, failed bool) {
	defer func() {
		if failed {
			value = recover()
		}
	}()
	failed = true
	run(ctx)
	return nil, false
}
