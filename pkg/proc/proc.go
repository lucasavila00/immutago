// Package proc runs supervised processes.
//
// A process is a mutable struct that embeds Process[A] and has a Run(context.Context)
// method with a pointer receiver. A is the immutable argument type of the process;
// use None for a process without arguments. Start creates a new process value,
// gives it the arguments, calls its Init method if it has one, runs Run in a new
// goroutine, and returns its PID. Callers
// keep the PID and call exported methods directly on the process pointer from
// PID.Proc. The immutago analyzer proves that these methods take and return only
// immutable values and PIDs. For processes that need each other's PIDs, Reserve
// a PID first and start it later with StartReserved.
//
// A supervisor runs its children until Stop. If Run returns, the process ends.
// If Run panics, the supervisor waits 1 s, doubling the delay for each restart
// in the last 5 minutes. It then creates a new process value from the same
// arguments, calls Init, stores the value in the same PID, and calls Run. Calls during the delay
// still use the old value; calls already in progress finish on that value. A sixth
// panic from the same child in 5 minutes stops all children of the supervisor in reverse start order and
// escalates the panic to the parent supervisor, which restarts the subtree with
// the same rule. At the root supervisor, the escalation panics again with the
// last panic value, and the program exits.
//
// Stop cancels each Run context and waits for Run to return. Cancellation is
// cooperative; Run must observe its context. Stop does not wait for direct
// method calls. Process methods must synchronize their own shared state.
package proc

import (
	"context"
	"slices"
	"sync"
	"sync/atomic"
	"time"

	"github.com/lucasavila00/immutago/pkg/im"
)

const (
	restartLimit  = 5
	restartWindow = 5 * time.Minute
)

// Process marks a mutable struct that runs as a supervised process with arguments of type A.
// Only Start and StartReserved create it, and other code calls its exported methods through a PID.
// Embed it by value. A named field or pointer embedding does not mark the struct.
type Process[A any] struct {
	args A
}

// Args returns the arguments that Start or StartReserved gave to the process.
func (p *Process[A]) Args() A { return p.args }

// bind sets the arguments of a new process value.
func (p *Process[A]) bind(args A) { p.args = args }

// None is the argument type of a process without arguments.
type None struct {
	im.Shared
}

// Supervisor runs processes and nested supervisors.
type Supervisor struct {
	mu       sync.Mutex
	children []*child
	ctx      context.Context
	failure  chan any
	stopping bool
	stopped  chan struct{}
	parent   *Supervisor
	entry    *child
}

// child is one process or nested supervisor and its current goroutine.
type child struct {
	run  func(context.Context) // nil while a permanent stop waits for the current watcher
	stop context.CancelFunc
	done chan struct{}
}

// PID identifies a process. Start returns a started PID; Reserve returns a PID to start later.
type PID[T any] struct {
	value *atomic.Pointer[T]
}

// Proc returns the current process value with one atomic load.
// It panics if the PID is not started.
func (p PID[T]) Proc() *T {
	value := p.value.Load()
	if value == nil {
		panic("proc: PID not started")
	}
	return value
}

// Reserve returns a PID that is not started. Proc panics until StartReserved starts it.
func Reserve[T any]() PID[T] { return PID[T]{value: new(atomic.Pointer[T])} }

// NewRoot returns a root supervisor.
func NewRoot() *Supervisor { return root(func(value any) { panic(value) }) }

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

// StartSupervisor returns a new supervisor that parent runs as a child.
// When the nested supervisor's Stop channel closes, its parent has removed its child entry.
func StartSupervisor(parent *Supervisor) *Supervisor {
	nested := &Supervisor{stopped: make(chan struct{}), parent: parent}
	nested.entry = parent.add(nested.serve)
	return nested
}

// Start reserves a PID and starts it with args under parent, as StartReserved does.
func Start[T, A any, P interface {
	*T
	Run(context.Context)
	bind(A)
}](parent *Supervisor, args A) PID[T] {
	pid := Reserve[T]()
	StartReserved[T, A, P](parent, pid, args)
	return pid
}

// StartReserved creates a process value with args, calls its Init method if it has one, stores it in pid, and
// runs its Run method under parent. A panic in the first Init reaches the caller.
// The PID is started when StartReserved returns. Each restart creates a new value from the same args.
// If parent is stopped, the process does not run. StartReserved panics if pid is already started.
func StartReserved[T, A any, P interface {
	*T
	Run(context.Context)
	bind(A)
}](parent *Supervisor, pid PID[T], args A) {
	create := func() *T {
		value := new(T)
		P(value).bind(args)
		if initializer, ok := any(value).(interface{ Init() }); ok {
			initializer.Init()
		}
		return value
	}
	if !pid.value.CompareAndSwap(nil, create()) {
		panic("proc: PID already started")
	}
	started := false
	parent.add(func(ctx context.Context) {
		if started {
			pid.value.Store(create())
		}
		started = true
		P(pid.value.Load()).Run(ctx)
	})
}

// Stop begins stopping; the channel closes when all children have stopped, last started first.
// Stop does not wait, so a process can stop its own supervisor from Run.
// For a nested supervisor, completion also means its parent has removed its child entry.
func (s *Supervisor) Stop() <-chan struct{} {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.stopping {
		s.stopping = true
		go func() {
			s.halt()
			if s.parent != nil {
				s.parent.release(s.entry)
				s.parent, s.entry = nil, nil
			}
			close(s.stopped)
		}()
	}
	return s.stopped
}

// add records a child, runs it if s is running, and returns it.
func (s *Supervisor) add(run func(context.Context)) *child {
	done := make(chan struct{})
	close(done)
	c := &child{run: run, stop: func() {}, done: done}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.children = append(s.children, c)
	if s.ctx != nil && !s.stopping {
		s.launch(c)
	}
	return c
}

// release cancels a child, prevents another launch, and removes it after its watcher returns.
func (s *Supervisor) release(c *child) {
	s.mu.Lock()
	c.run = nil
	stop, done := c.stop, c.done
	s.mu.Unlock()
	stop()
	<-done
	s.mu.Lock()
	defer s.mu.Unlock()
	// Keep the child order stable for any halt already reading a snapshot.
	s.children = slices.DeleteFunc(slices.Clone(s.children), func(other *child) bool { return other == c })
}

// launch runs c in a new goroutine. The caller holds s.mu.
func (s *Supervisor) launch(c *child) {
	if c.run == nil {
		return
	}
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

// attempt calls run with a context that ends with the attempt, and returns the panic value if run panics.
func attempt(ctx context.Context, run func(context.Context)) (value any, failed bool) {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	defer func() {
		if failed {
			value = recover()
		}
	}()
	failed = true
	run(ctx)
	return nil, false
}
