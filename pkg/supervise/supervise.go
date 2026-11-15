// Package supervise manages background services and restarts them after a panic.
//
// Embed Service[A], supply Run(context.Context), and optionally declare Init() error.
// Start initializes the service and returns Ref[T]. Call ref.Borrow().Method(args)
// to select the current service after any restart. Calls run in the caller's
// goroutine; review shared fields, locks, saved inputs, and dependency calls in
// the service code. The checker validates exported results and direct Borrow calls.
//
// The service guide is at
// https://github.com/lucasavila00/immutago/blob/main/docs/guides/services.md.
// The complete source rules and runtime behavior are at
// https://github.com/lucasavila00/immutago/blob/main/docs/reference/services.md.
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
// Embed Service[A] to supply start arguments and runtime methods; add
// Run(context.Context) for background work. Start and StartReserved create and
// initialize each service value.
// Call an exported service method with ref.Borrow().Method(args).
type Service[A any] struct {
	args     A
	report   func(value any, stack []byte)
	mu       sync.Mutex
	control  context.Context
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
// The runtime calls Init on each fresh value before Borrow or Run can use it.
// A declared Init must release any application resources it opened before
// returning an error; the runtime stops staged children.
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
		if p.failure, _, p.failed = attempt(context.Background(), func(ctx context.Context) {
			p.children.serve(p.control, ctx)
		}, false); p.failed {
			p.cancel()
		}
	}()
}

// bind sets the arguments and panic reporter before the service is published.
func (p *Service[A]) bind(args A, report func(any, []byte)) {
	p.args, p.report = args, report
}

// begin starts child supervision and returns the context of the run.
// control limits retries. When ctx ends, halt stops children before it cancels Run.
func (p *Service[A]) begin(control, ctx context.Context) context.Context {
	p.mu.Lock()
	defer p.mu.Unlock()
	run, cancel := context.WithCancel(context.WithoutCancel(ctx))
	p.control = control
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

// end records whether shutdown had started, then stops and waits for children.
// A child failure still fails this attempt, including after a normal Run return.
func (p *Service[A]) end() runOutcome {
	if p.unwatch != nil {
		p.unwatch()
	}
	p.mu.Lock()
	outcome := completed
	if p.ended {
		outcome = interrupted
	}
	p.ended = true
	if p.cancel != nil {
		p.cancel()
	}
	children, done := p.children, p.done
	p.mu.Unlock()
	if children == nil {
		return outcome
	}
	<-children.Stop()
	if done != nil {
		<-done
	}
	if p.failed {
		panic(p.failure)
	}
	return outcome
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
	mu          sync.Mutex
	children    []*child
	ctx         context.Context
	control     context.Context
	stopControl context.CancelFunc
	failure     chan any
	stopping    bool
	stopped     chan struct{}
	parent      *Group
	entry       *child
	report      func(value any, stack []byte)
}

// child is one service or nested group. A nil done marks pending setup.
// Temporary halt preserves pending setup; Stop ends it. A started watcher has done.
type child struct {
	run         func(context.Context, context.Context) runOutcome // nil while a permanent stop waits for the current watcher
	stop        context.CancelFunc
	stopControl context.CancelFunc
	done        chan struct{} // nil until the first watcher starts
	group       *Group
}

// runOutcome states whether a child has completed or can run in a later group run.
type runOutcome uint8

const (
	interrupted runOutcome = iota
	completed
)

// Ref is a reference to one service. Copies select the same service, including
// after a restart. The analyzer requires T to be a concrete struct that embeds Service.
// References can be stored in immutable fields and returned as service results;
// the service state can still change.
// Start returns an initialized reference. Reserve creates one for later startup;
// a zero Ref is unset and cannot be used in place of Reserve. Use IsZero to test it.
type Ref[T any] struct {
	value *atomic.Pointer[T]
}

// IsZero reports whether the reference is the unset zero Ref. Borrow on a zero
// Ref panics. IsZero does not check service readiness: a reference from Reserve
// is nonzero before start, and a started reference stays nonzero after stop.
func (p Ref[T]) IsZero() bool { return p.value == nil }

// Borrow selects the last successfully initialized service value with one atomic load.
// For direct Ref calls, the analyzer accepts only ref.Borrow().Method(args).
// Method must be exported and must not be Init, Run, or a runtime method.
// Direct calls cannot save the pointer, Borrow method, or selected method.
// Interface calls can expose the pointer without these source checks.
// Borrow is shared and is not exclusive. No release call is needed.
// A call in progress finishes on its selected instance, even after a restart.
//
// Borrow panics for a zero or unstarted reference. It can select a value whose
// Run has ended. Stop does not invalidate the reference or block later calls.
func (p Ref[T]) Borrow() *T {
	value := p.value.Load()
	if value == nil {
		panic("supervise: Ref not started")
	}
	return value
}

// Reserve creates a reference that StartReserved can initialize later. Use it
// to connect services whose start arguments need each other's references.
// Borrow panics until initialization succeeds.
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
		if value, _, failed := attempt(context.Background(), func(ctx context.Context) {
			s.serve(context.Background(), ctx)
		}, false); failed {
			s.mu.Lock()
			defer s.mu.Unlock()
			if !s.stopping {
				exit(value)
			}
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
	parent.add(func(control, ctx context.Context) runOutcome {
		nested.serve(control, ctx)
		return interrupted
	}, nested.halt, nested)
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
	begin(context.Context, context.Context) context.Context
	end() runOutcome
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
	begin(context.Context, context.Context) context.Context
	end() runOutcome
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
	parent.add(func(control, ctx context.Context) (outcome runOutcome) {
		value := first.Swap(nil)
		if value == nil {
			if control.Err() != nil || ctx.Err() != nil {
				return interrupted
			}
			var err error
			value, err = create()
			if err != nil {
				panic(err)
			}
		}
		service := P(value)
		defer func() {
			ended := service.end()
			if outcome == completed {
				outcome = ended
			}
		}()
		if control.Err() != nil || ctx.Err() != nil {
			return interrupted
		}
		ref.value.Store(value)
		service.Run(service.begin(control, ctx))
		return completed
	}, func() {
		if value := first.Swap(nil); value != nil {
			P(value).end()
		}
	}, nil)
	return nil
}

// Stop prevents new retries in the subtree, then requests ordered shutdown.
// It returns its completion channel without waiting.
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
		if s.stopControl != nil {
			s.stopControl()
		}
		if s.parent != nil {
			s.parent.mu.Lock()
			stopControl := s.entry.stopControl
			s.parent.mu.Unlock()
			if stopControl != nil {
				stopControl()
			}
		}
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
func (s *Group) add(run func(context.Context, context.Context) runOutcome, stop func(), group *Group) {
	c := &child{run: run, stop: stop, group: group}
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
	if done != nil {
		<-done
	}
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
	control, stopControl := context.WithCancel(s.control)
	done := make(chan struct{})
	c.stop, c.done = cancel, done
	c.stopControl = stopControl
	run := c.run
	go func() {
		defer close(done)
		defer stopControl()
		if watch(control, ctx, run, s.failure, s.report) == completed {
			s.forget(c)
		}
	}()
}

// serve runs one incarnation of s. control governs supervision in the subtree;
// ctx ends this incarnation. Cancellation of control does not change stop order.
func (s *Group) serve(control, ctx context.Context) {
	s.mu.Lock()
	if s.stopping || control.Err() != nil {
		s.mu.Unlock()
		return
	}
	failure := make(chan any, 1)
	policy, cancelPolicy := context.WithCancel(control)
	s.control, s.stopControl = policy, cancelPolicy
	s.ctx, s.failure = context.WithoutCancel(ctx), failure
	for _, c := range s.children {
		s.launch(c)
	}
	s.mu.Unlock()
	var value any
	failed := false
	for {
		select {
		case <-ctx.Done():
		case <-s.stopped:
		case value = <-failure:
			if policy.Err() != nil {
				failure = nil
				continue
			}
			failed = true
		}
		break
	}
	s.mu.Lock()
	s.ctx = nil
	cancelPolicy()
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
		if done == nil && !permanent {
			continue
		}
		if permanent && c.group != nil {
			<-c.group.Stop()
		} else {
			stop()
			if done != nil {
				<-done
			}
		}
	}
}

// watch retries failed attempts while control and ctx permit another attempt.
// A completed result stays completed when cancellation occurs during cleanup.
func watch(control, ctx context.Context, run func(context.Context, context.Context) runOutcome, failure chan<- any,
	report func(value any, stack []byte),
) runOutcome {
	var restarts []time.Time
	for {
		outcome := interrupted
		value, stack, failed := attempt(ctx, func(ctx context.Context) {
			outcome = run(control, ctx)
		}, report != nil)
		if failed && report != nil {
			report(value, stack)
		}
		if !failed {
			return outcome
		}
		if control.Err() != nil || ctx.Err() != nil {
			return interrupted
		}
		now := time.Now()
		restarts = slices.DeleteFunc(restarts, func(at time.Time) bool { return now.Sub(at) >= restartWindow })
		if len(restarts) == restartLimit {
			select {
			case failure <- value:
			default:
			}
			return interrupted
		}
		delay := time.NewTimer(time.Second << len(restarts))
		restarts = append(restarts, now)
		select {
		case <-delay.C:
		case <-control.Done():
			delay.Stop()
			return interrupted
		case <-ctx.Done():
			delay.Stop()
			return interrupted
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
