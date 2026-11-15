// Package supervise manages background services and restarts them after a panic.
//
// Define a service:
//   - Embed Service[A] for start arguments of type A; use NoArgs for no arguments.
//   - Declare Run(context.Context) on a pointer receiver for background work.
//   - Declare Init() error for setup before callers can use the service.
//   - Start calls Init, returns Ref[T], and schedules Run in a managed goroutine.
//
// Call a service:
//   - Keep the Ref and use ref.Current().Method().
//   - Calls run in the caller's goroutine and can overlap with Run and other calls.
//     Synchronize access to mutable fields.
//   - Each exported method except Init and Run returns exactly one Scoped result:
//     an immutable value such as an integer, a marked struct, or a frozen collection.
//     The analyzer rejects returned pointers and mutable result types.
//   - Arguments and implementation code follow ordinary Go rules. Uses of im
//     values keep their immutability and pointer rules.
//
// Review state changes, locks, saved inputs, and dependency calls in the service
// code. Immutable results do not prove service isolation.
//
// # Groups and shutdown
//
//   - NewRoot creates an application group; StartGroup creates a nested group.
//   - Service.Children supplies the group for one run. Children created during
//     Init wait for Run to begin.
//   - Group.Stop returns a completion channel. Children stop last started first,
//     before their parent's Run context is cancelled.
//   - Run must return when its context ends. Stop cannot force it to return and
//     does not wait for direct method calls. Init has no cancellation context.
//
// # Replacement and failure
//
//   - Normal Run return ends the service. A panic starts a delayed restart.
//   - Delay starts at 1 second and doubles for each restart in the last 5 minutes.
//   - Each attempt creates a fresh value with the same arguments and calls Init.
//     Successful setup replaces the current value; failed setup counts as a failure.
//   - Six failures from one child in 5 minutes stop its siblings and pass failure
//     to the parent group. The parent restarts that group under the same rules.
//     Failure at the root panics and ends the program.
//   - NewRoot can report recovered panics and restart Init errors before each delay.
//   - Current selects the last published value, even during delays and after stop.
//     Calls in progress finish on their selected value.
//   - A direct call's panic stays in its goroutine. Calls from Run or a restart
//     Init can fail that attempt. The first Init panic reaches the Start caller.
//   - A child Ref does not follow its parent's restart.
package supervise

import (
	"context"
	"runtime/debug"
	"slices"
	"sync"
	"sync/atomic"
	"time"
)

const (
	restartLimit  = 5
	restartWindow = 5 * time.Minute
)

// Service supplies the arguments and child group of a supervised service.
// Embed one Service[A] by value in a named, non-generic struct, and declare
// Run(context.Context) with a pointer receiver. Start and StartReserved create
// the struct; callers use a Ref instead of constructing or retaining its pointer.
type Service[A any] struct {
	args     A
	report   func(value any, stack []byte)
	mu       sync.Mutex
	cancel   context.CancelFunc
	unwatch  func() bool
	children *Group
	done     chan struct{}
	ended    bool
	failure  any
	failed   bool
}

// Args reads the start arguments of this service value. Each restart receives
// the same arguments.
func (p *Service[A]) Args() A { return p.args }

// Init supplies successful empty setup when the service does not declare Init.
// The runtime calls Init on each fresh value before Current or Run can use it.
// A declared Init must release any application
// resources it opened before returning an error; the runtime stops staged children.
func (*Service[A]) Init() error { return nil }

// Children returns the group owned by this service run. Use it as the parent
// of Start, StartReserved, or StartGroup. The runtime stops it when this run ends.
// To stop selected children separately, create a nested group.
//
// Each run has a separate child group. Children created before Run begins wait
// for that run. Failed setup stops them without running them. On stop, the runtime
// waits for children, last started first, before it cancels this service's Run
// context. A return or panic from Run also stops the children before the run ends.
//
// A child that passes its restart limit fails this run. After the run ends,
// Children returns a stopped group; later starts can initialize but cannot run.
func (p *Service[A]) Children() *Group {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.children == nil {
		p.children = &Group{stopped: make(chan struct{}), report: p.report}
		switch {
		case p.ended:
			p.children.Stop()
		case p.cancel != nil:
			p.serve()
		}
	}
	return p.children
}

// serve runs the group of the run until it stops, and ends the run if a child escalates.
// The caller holds p.mu.
func (p *Service[A]) serve() {
	p.done = make(chan struct{})
	go func() {
		defer close(p.done)
		if p.failure, _, p.failed = attempt(context.Background(), p.children.serve, false); p.failed {
			p.cancel()
		}
	}()
}

// bind sets the arguments and panic reporter before the service is published.
func (p *Service[A]) bind(args A, report func(any, []byte)) {
	p.args, p.report = args, report
}

// begin returns the context of the run and runs the group of the run if it exists.
// When ctx ends, halt stops the children before the context of the run ends.
func (p *Service[A]) begin(ctx context.Context) context.Context {
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

// halt stops the group of the run and waits for it, then ends the context of the run.
func (p *Service[A]) halt() {
	p.mu.Lock()
	p.ended = true
	children := p.children
	p.mu.Unlock()
	if children != nil {
		<-children.Stop()
	}
	p.cancel()
}

// end stops the group of the run and waits for it, and panics if a child escalated to it.
func (p *Service[A]) end() {
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

// NoArgs supplies an empty start argument. Embed Service[NoArgs] and pass
// NoArgs{} to Start when the service needs no arguments.
type NoArgs struct{}

// ConcurrentSafe carries a caller's promise that its value is safe for concurrent use.
// The wrapper adds no synchronization and has no special checker rule.
// Ordinary service arguments can hold T directly.
type ConcurrentSafe[T any] struct {
	value T
}

// AssumeConcurrentSafe records the caller's promise that concurrent use of v
// is safe. It adds no checking, synchronization, or immutability to v.
func AssumeConcurrentSafe[T any](v T) ConcurrentSafe[T] { return ConcurrentSafe[T]{value: v} }

// Value reads the wrapped dependency. A zero ConcurrentSafe value contains the
// zero value of T.
func (d ConcurrentSafe[T]) Value() T { return d.value }

// Group manages the lifetime and restart rules of its child services and groups.
// Create a root with NewRoot or a nested group with StartGroup. The zero value
// is not usable. A group returned by Service.Children belongs to that service run.
type Group struct {
	mu       sync.Mutex
	children []*child
	ctx      context.Context
	failure  chan any
	stopping bool
	stopped  chan struct{}
	parent   *Group
	entry    *child
	report   func(value any, stack []byte)
}

// child is one service or nested group and its current goroutine.
type child struct {
	run   func(context.Context) // nil while a permanent stop waits for the current watcher
	stop  context.CancelFunc
	done  chan struct{}
	group *Group
}

// Ref is a typed reference to the current value of one service. Copies refer
// to the same service, including after its restart replaces the value.
// Start returns an initialized reference. Reserve creates one for later startup;
// a zero Ref is empty and cannot be used in place of Reserve.
type Ref[T any] struct {
	value *atomic.Pointer[T]
}

// Current selects the last successfully initialized service value with one
// atomic load. A saved pointer keeps selecting that value after a restart;
// call Current again to select the replacement.
//
// Current panics for an empty or unstarted reference. It can return a value whose
// Run has ended. Stop does not invalidate the reference or block later calls.
func (p Ref[T]) Current() *T {
	value := p.value.Load()
	if value == nil {
		panic("supervise: Ref not started")
	}
	return value
}

// Reserve creates a reference that StartReserved can initialize later. Use it
// to connect services whose start arguments need each other's references.
// Current panics until initialization succeeds.
func Reserve[T any]() Ref[T] { return Ref[T]{value: new(atomic.Pointer[T])} }

// NewRoot creates a running root group. A failure that reaches this group
// panics and ends the program.
//
// If report is non-nil, it receives each recovered panic or restart Init error
// and a stack before the restart delay. Child groups use the same function.
// Reports for different children can overlap. The group waits for report to
// return, and a panic in report is not recovered. Do not wait for that group
// to stop inside report.
func NewRoot(report func(value any, stack []byte)) *Group {
	return root(func(value any) { panic(value) }, report)
}

// root returns a root group that calls exit with an escalated panic value.
func root(exit func(any), report func(value any, stack []byte)) *Group {
	s := &Group{stopped: make(chan struct{}), report: report}
	go func() {
		if value, _, failed := attempt(context.Background(), s.serve, false); failed {
			exit(value)
		}
	}()
	return s
}

// StartGroup creates a child group under parent. Use it to stop selected
// services together or restart them together when a child passes its restart limit.
// A stopped parent returns a stopped group. When this group's Stop channel
// closes, the parent has also removed it from the child list.
func StartGroup(parent *Group) *Group {
	nested := &Group{stopped: make(chan struct{}), parent: parent, report: parent.report}
	parent.add(nested.serve, nested.halt, nested)
	return nested
}

// Start creates T with args, calls Init, and returns its reference after setup
// succeeds. Run then runs under parent. Only T must be written as a type argument.
//
// An Init error returns an empty Ref and that error. An Init panic reaches the
// caller. If parent has stopped, setup still runs but Run does not. Start uses
// Reserve and StartReserved; their initialization and restart rules apply.
func Start[T, A any, P interface {
	*T
	Init() error
	Run(context.Context)
	bind(A, func(any, []byte))
	begin(context.Context) context.Context
	end()
}](parent *Group, args A) (Ref[T], error) {
	ref := Reserve[T]()
	if err := StartReserved[T, A, P](parent, ref, args); err != nil {
		return Ref[T]{}, err
	}
	return ref, nil
}

// StartReserved starts a reference from Reserve under parent. It sets args on
// a fresh T and calls Init before publishing T or starting Run. An Init error
// leaves ref unstarted, so the caller can retry. An Init panic reaches the caller.
//
// Each restart repeats setup on a fresh value with the same args. Failed setup
// keeps the previous value in ref. If parent has stopped, Init still runs but
// Run does not. The runtime stops children created by a value that will not run.
//
// Starting an already started reference panics. If concurrent starts compete,
// the one that loses publication stops its staged children and panics.
func StartReserved[T, A any, P interface {
	*T
	Init() error
	Run(context.Context)
	bind(A, func(any, []byte))
	begin(context.Context) context.Context
	end()
}](parent *Group, ref Ref[T], args A) error {
	if ref.value.Load() != nil {
		panic("supervise: Ref already started")
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
	if !ref.value.CompareAndSwap(nil, value) {
		P(value).end()
		panic("supervise: Ref already started")
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
		service := P(value)
		defer service.end()
		if ctx.Err() != nil {
			return
		}
		ref.value.Store(value)
		service.Run(service.begin(ctx))
	}, func() {
		if value := first.Swap(nil); value != nil {
			P(value).end()
		}
	}, nil)
	return nil
}

// Stop requests shutdown and returns its completion channel without waiting.
// Repeated calls return the same channel. The channel closes after all children
// stop, last started first, and a nested group is removed from its parent.
//
// Run methods must return when their contexts end. Stop does not wait for direct
// service method calls, and their references remain callable. A Run method can
// request its own group's stop, but must not wait for that stop to complete.
func (s *Group) Stop() <-chan struct{} {
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
func (s *Group) add(run func(context.Context), stop func(), group *Group) {
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
func (s *Group) release(c *child) {
	s.mu.Lock()
	c.run = nil
	stop, done := c.stop, c.done
	s.mu.Unlock()
	stop()
	<-done
	s.forget(c)
}

// forget prevents another launch of a child and removes it.
func (s *Group) forget(c *child) {
	s.mu.Lock()
	defer s.mu.Unlock()
	c.run = nil
	// Keep the child order stable for any halt already reading a snapshot.
	s.children = slices.DeleteFunc(slices.Clone(s.children), func(other *child) bool { return other == c })
}

// launch runs c in a new goroutine. The caller holds s.mu.
func (s *Group) launch(c *child) {
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
func (s *Group) serve(ctx context.Context) {
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
func (s *Group) halt() {
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
