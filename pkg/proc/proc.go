// Package proc runs supervised processes.
//
// A process is a mutable struct that embeds Process[A] and has a Run(context.Context)
// method with a pointer receiver. A holds immutable arguments or explicit unchecked
// dependencies; use None for a process without arguments. Start creates a new process value,
// gives it the arguments, runs Run in a new goroutine, and returns its PID. Callers
// keep the PID and call exported methods directly on the process pointer from
// PID.Proc. The immutago analyzer proves that these methods take and return only
// immutable values and PIDs. For processes that need each other's PIDs, Reserve
// a PID first and start it later with StartReserved.
//
// A supervisor runs its children until Stop. If Run returns, the process ends, and the supervisor removes it.
// If Run panics, the supervisor waits 1 s, doubling the delay for each restart
// in the last 5 minutes. It then creates a new process value from the same
// arguments, stores it in the same PID, and calls Run. Calls during the delay
// still use the old value; calls already in progress finish on that value. A sixth
// panic from the same child in 5 minutes stops all children of the supervisor in reverse start order and
// escalates the panic to the parent supervisor, which restarts the subtree with
// the same rule. At the root supervisor, the escalation panics again with the
// last panic value, and the program exits. NewRoot accepts a function to report recovered panics.
//
// A process can start children under the Supervisor of its current run. When
// the process stops, the runtime stops these children in reverse start order
// before the Run context ends, so children can still call the process. When
// Run returns or panics, the runtime stops them before the process restarts or
// ends. A child that escalates fails the run.
//
// Stop cancels each Run context and waits for Run to return. Cancellation is
// cooperative; Run must observe its context. Stop does not wait for direct
// method calls. Process methods must synchronize their own shared state.
package proc

import (
	"context"
	"runtime/debug"
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
	args     A
	mu       sync.Mutex
	cancel   context.CancelFunc
	unwatch  func() bool
	children *Supervisor
	done     chan struct{}
	ended    bool
	failure  any
	failed   bool
	report   func(value any, stack []byte)
}

// Args returns the arguments that Start or StartReserved gave to the process.
func (p *Process[A]) Args() A { return p.args }

// Supervisor returns the supervisor of the current run of the process. Each run has its own supervisor.
// Processes started under it before Run begins start with the run. When the process stops, the runtime
// stops its children, last started first, and waits for them before the Run context ends. When Run returns
// or panics, the runtime stops them and waits for them before the run ends. After the run ends,
// Supervisor returns a stopped supervisor, and processes started under it do not run.
// If a child escalates, the run context ends and the run fails with the panic value of the child.
func (p *Process[A]) Supervisor() *Supervisor {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.children == nil {
		p.children = &Supervisor{stopped: make(chan struct{}), report: p.report}
		switch {
		case p.ended:
			p.children.Stop()
		case p.cancel != nil:
			p.serve()
		}
	}
	return p.children
}

// serve runs the supervisor of the run until it stops, and ends the run if a child escalates.
// The caller holds p.mu.
func (p *Process[A]) serve() {
	p.done = make(chan struct{})
	go func() {
		defer close(p.done)
		if p.failure, _, p.failed = attempt(context.Background(), p.children.serve, false); p.failed {
			p.cancel()
		}
	}()
}

// bind sets the arguments of a new process value and the panic report of its run supervisors.
func (p *Process[A]) bind(args A, report func(value any, stack []byte)) {
	p.args, p.report = args, report
}

// begin returns the context of the run and runs the supervisor of the run if it exists.
// When ctx ends, halt stops the children before the context of the run ends.
func (p *Process[A]) begin(ctx context.Context) context.Context {
	p.mu.Lock()
	defer p.mu.Unlock()
	run, cancel := context.WithCancel(context.WithoutCancel(ctx))
	p.cancel = cancel
	p.unwatch = context.AfterFunc(ctx, p.halt)
	if p.children != nil {
		p.serve()
	}
	return run
}

// halt stops the supervisor of the run and waits for it, then ends the context of the run.
func (p *Process[A]) halt() {
	p.mu.Lock()
	p.ended = true
	children := p.children
	p.mu.Unlock()
	if children != nil {
		<-children.Stop()
	}
	p.cancel()
}

// end stops the supervisor of the run and waits for it, and panics if a child escalated to it.
func (p *Process[A]) end() {
	p.unwatch()
	p.mu.Lock()
	p.ended = true
	p.cancel()
	children, done := p.children, p.done
	p.mu.Unlock()
	if children == nil {
		return
	}
	<-children.Stop()
	if done != nil {
		<-done
	}
	if p.failed {
		panic(p.failure)
	}
}

// None is the argument type of a process without arguments.
type None struct {
	im.Shared
}

// UncheckedDependency holds a value that the caller trusts to be safe for concurrent use by several processes,
// for example a logger, an HTTP client, a factory function, or a test fake. This does not make the value immutable.
// The analyzer does not prove this trust. It permits UncheckedDependency only as a process argument type or as a
// named field of a marked process argument struct.
type UncheckedDependency[T any] struct {
	value T
}

// Trust wraps v. The call is the caller's promise that several processes can use v at the same time.
func Trust[T any](v T) UncheckedDependency[T] { return UncheckedDependency[T]{value: v} }

// Value returns the trusted value, or the zero value of T for a zero dependency.
// The analyzer permits only direct calls on the process receiver's Args() or one of its dependency fields.
func (d UncheckedDependency[T]) Value() T { return d.value }

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
	report   func(value any, stack []byte)
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

// NewRoot returns a root supervisor. Pass nil to disable panic reports.
// Otherwise, report receives each recovered panic value and stack before the restart delay.
// Nested supervisors inherit report. Calls can run concurrently, and the supervisor waits
// for report to return. A panic in report is not recovered.
func NewRoot(report func(value any, stack []byte)) *Supervisor {
	return root(func(value any) { panic(value) }, report)
}

// root returns a root supervisor that calls exit with an escalated panic value.
func root(exit func(any), report func(value any, stack []byte)) *Supervisor {
	s := &Supervisor{stopped: make(chan struct{}), report: report}
	go func() {
		if value, _, failed := attempt(context.Background(), s.serve, false); failed {
			exit(value)
		}
	}()
	return s
}

// StartSupervisor returns a new supervisor that parent runs as a child.
// When the nested supervisor's Stop channel closes, its parent has removed its child entry.
func StartSupervisor(parent *Supervisor) *Supervisor {
	nested := &Supervisor{stopped: make(chan struct{}), parent: parent, report: parent.report}
	nested.entry = parent.add(nested.serve)
	return nested
}

// Start reserves a PID and starts it with args under parent, as StartReserved does.
func Start[T, A any, P interface {
	*T
	Run(context.Context)
	bind(A, func(value any, stack []byte))
	begin(context.Context) context.Context
	end()
}](parent *Supervisor, args A) PID[T] {
	pid := Reserve[T]()
	StartReserved[T, A, P](parent, pid, args)
	return pid
}

// StartReserved creates a process value with args, stores it in pid, and runs its Run method under parent.
// The PID is started when StartReserved returns. Each restart creates a new value from the same args.
// If parent is stopped, the process does not run. StartReserved panics if pid is already started.
func StartReserved[T, A any, P interface {
	*T
	Run(context.Context)
	bind(A, func(value any, stack []byte))
	begin(context.Context) context.Context
	end()
}](parent *Supervisor, pid PID[T], args A) {
	create := func() *T {
		value := new(T)
		P(value).bind(args, parent.report)
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
		value := P(pid.value.Load())
		defer value.end()
		value.Run(value.begin(ctx))
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
	s.forget(c)
}

// forget prevents another launch of a child and removes it.
func (s *Supervisor) forget(c *child) {
	s.mu.Lock()
	defer s.mu.Unlock()
	c.run = nil
	// Keep the child order stable for any halt already reading a snapshot.
	s.children = slices.DeleteFunc(slices.Clone(s.children), func(other *child) bool { return other == c })
}

// launch runs c in a new goroutine. The caller holds s.mu.
func (s *Supervisor) launch(c *child) {
	if c.run == nil {
		return
	}
	ctx, cancel := context.WithCancel(s.ctx)
	done := make(chan struct{})
	c.stop, c.done = cancel, done
	run := c.run
	go func() {
		defer close(done)
		if watch(ctx, run, s.failure, s.report) {
			s.forget(c)
		}
	}()
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
// It reports whether run returned before ctx ended.
func watch(ctx context.Context, run func(context.Context), failure chan<- any,
	report func(value any, stack []byte),
) bool {
	var restarts []time.Time
	for {
		value, stack, failed := attempt(ctx, run, report != nil)
		if failed && report != nil {
			report(value, stack)
		}
		if !failed || ctx.Err() != nil {
			return !failed && ctx.Err() == nil
		}
		now := time.Now()
		restarts = slices.DeleteFunc(restarts, func(at time.Time) bool { return now.Sub(at) >= restartWindow })
		if len(restarts) == restartLimit {
			select {
			case failure <- value:
			default:
			}
			return false
		}
		delay := time.NewTimer(time.Second << len(restarts))
		restarts = append(restarts, now)
		select {
		case <-delay.C:
		case <-ctx.Done():
			delay.Stop()
			return false
		}
	}
}

// attempt calls run with a context that ends with the attempt, and returns the panic value and its stack if run
// panics.
func attempt(ctx context.Context, run func(context.Context), captureStack bool) (value any, stack []byte, failed bool) {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	defer func() {
		if failed {
			value = recover()
			if captureStack {
				stack = debug.Stack()
			}
		}
	}()
	failed = true
	run(ctx)
	return nil, nil, false
}
