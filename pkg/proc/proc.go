// Package proc runs supervised processes.
//
// A process is a mutable struct that embeds Process[A] and has a Run(context.Context)
// method with a pointer receiver. A holds immutable arguments or explicit unchecked
// dependencies; use None for a process without arguments. Start creates a new process value,
// gives it the arguments, calls Init, and returns an error if Init fails. After Init succeeds,
// Start publishes the value, runs Run in a new goroutine, and returns its PID. Callers
// keep the PID and call exported methods directly on the process pointer from
// PID.Proc. The immutago analyzer proves that these methods take and return only
// immutable values and PIDs. For processes that need each other's PIDs, Reserve
// a PID first and start it later with StartReserved.
//
// A supervisor runs its children until Stop. If Run returns, the process ends, and the supervisor removes it.
// If Run panics, the supervisor waits 1 s, doubling the delay for each restart
// in the last 5 minutes. It then creates a new process value from the same
// arguments, calls Init, stores it in the same PID, and calls Run. An Init error or panic
// fails that restart attempt and keeps the old value in the PID. Calls during the delay
// still use the old value; calls already in progress finish on that value. A sixth
// failed run or initialization from the same child in 5 minutes stops all children of the supervisor in reverse
// start order and escalates the failure to the parent supervisor, which restarts the subtree with
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
	report   func(value any, stack []byte)
	mu       sync.Mutex
	cancel   context.CancelFunc
	unwatch  func() bool
	children *Supervisor
	done     chan struct{}
	ended    bool
	failure  any
	failed   bool
}

// Args returns the arguments that Start or StartReserved gave to the process.
func (p *Process[A]) Args() A { return p.args }

// Init is the default initialization method. It succeeds without further setup.
// The runtime calls Init before publication and before each restart; user code must not call it.
func (*Process[A]) Init() error { return nil }

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

// bind sets the arguments and panic reporter before the process is published.
func (p *Process[A]) bind(args A, report func(any, []byte)) {
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
	if p.unwatch != nil {
		p.unwatch()
	}
	p.mu.Lock()
	p.ended = true
	if p.cancel != nil {
		p.cancel()
	}
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

// ConcurrentSafe holds a value that the caller promises is safe for concurrent use.
// The analyzer does not check this promise. It permits ConcurrentSafe only as a
// process argument type or a named field of a marked process argument struct.
type ConcurrentSafe[T any] struct {
	value T
}

// AssumeConcurrentSafe wraps v with the caller's promise that concurrent use is safe.
// It does not check, freeze, or synchronize v.
func AssumeConcurrentSafe[T any](v T) ConcurrentSafe[T] { return ConcurrentSafe[T]{value: v} }

// Value returns the trusted value, or the zero value of T for a zero dependency.
// The analyzer permits only direct calls on the process receiver's Args() or one of its dependency fields.
func (d ConcurrentSafe[T]) Value() T { return d.value }

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
	run   func(context.Context) // nil while a permanent stop waits for the current watcher
	stop  context.CancelFunc
	done  chan struct{}
	group *Supervisor
}

// PID identifies a process. A successful Start returns a started PID; Reserve returns a PID to start later.
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
// Otherwise, report receives each recovered panic or restart Init error with a stack before the restart delay.
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
	parent.add(nested.serve, nested.halt, nested)
	return nested
}

// Start reserves a PID and starts it with args under parent, as StartReserved does.
// If Init returns an error, Start returns an empty PID and that error. An Init panic propagates to the caller.
func Start[T, A any, P interface {
	*T
	Init() error
	Run(context.Context)
	bind(A, func(any, []byte))
	begin(context.Context) context.Context
	end()
}](parent *Supervisor, args A) (PID[T], error) {
	pid := Reserve[T]()
	if err := StartReserved[T, A, P](parent, pid, args); err != nil {
		return PID[T]{}, err
	}
	return pid, nil
}

// StartReserved creates a process value with args and calls Init before it publishes the value or starts Run.
// If Init returns an error, the PID stays unstarted and can be retried. An Init panic propagates to the caller.
// Each restart calls Init on a new value with the same args. A failed Init keeps the previous value in the PID.
// If parent is stopped, Init still runs but Run does not. Staged children are stopped if the value will not run.
// StartReserved panics if pid is already started; a losing concurrent start stops its staged children.
func StartReserved[T, A any, P interface {
	*T
	Init() error
	Run(context.Context)
	bind(A, func(any, []byte))
	begin(context.Context) context.Context
	end()
}](parent *Supervisor, pid PID[T], args A) error {
	if pid.value.Load() != nil {
		panic("proc: PID already started")
	}
	create := func() (*T, error) {
		value := new(T)
		P(value).bind(args, parent.report)
		ready := false
		defer func() {
			if !ready {
				P(value).end()
			}
		}()
		if err := P(value).Init(); err != nil {
			return nil, err
		}
		ready = true
		return value, nil
	}
	value, err := create()
	if err != nil {
		return err
	}
	if !pid.value.CompareAndSwap(nil, value) {
		P(value).end()
		panic("proc: PID already started")
	}
	var first atomic.Pointer[T]
	first.Store(value)
	parent.add(func(ctx context.Context) {
		value := first.Swap(nil)
		if value == nil {
			if ctx.Err() != nil {
				return
			}
			var err error
			value, err = create()
			if err != nil {
				panic(err)
			}
		}
		process := P(value)
		defer process.end()
		if ctx.Err() != nil {
			return
		}
		pid.value.Store(value)
		process.Run(process.begin(ctx))
	}, func() {
		if value := first.Swap(nil); value != nil {
			P(value).end()
		}
	}, nil)
	return nil
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

// add records a child unless s is stopping and runs it if s is running.
// stop cleans up a child that has not run; launch replaces it with context cancellation.
func (s *Supervisor) add(run func(context.Context), stop func(), group *Supervisor) {
	done := make(chan struct{})
	close(done)
	c := &child{run: run, stop: stop, done: done, group: group}
	if group != nil {
		group.entry = c
	}
	s.mu.Lock()
	if s.stopping {
		s.mu.Unlock()
		if group != nil {
			<-group.Stop()
		} else {
			stop()
		}
		return
	}
	defer s.mu.Unlock()
	s.children = append(s.children, c)
	if s.ctx != nil {
		s.launch(c)
	}
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
		permanent := s.stopping
		s.mu.Unlock()
		if permanent && c.group != nil {
			<-c.group.Stop()
		} else {
			stop()
			<-done
		}
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
