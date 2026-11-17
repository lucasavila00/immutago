// Package supervise runs services with owned child lifetimes and terminal failures.
//
// Embed Service[A], supply Run(context.Context), and optionally declare Init(*Supervisor) error.
// Start initializes the service and returns Ref[T]. Calls through Borrow run in
// the caller's goroutine; review shared fields, locks, and inputs in service code.
// A Run failure stops the root. Supervisors control lifetime and stop order.
// See https://github.com/lucasavila00/immutago/blob/main/pkg/supervise/contract.md for the runtime contract.
package supervise

import (
	"context"
	"errors"
	"reflect"
	"runtime/debug"
	"sync"

	"github.com/lucasavila00/immutago/pkg/im"
)

var errGoexit = errors.New("supervise: service called runtime.Goexit")

// noCopy lets go vet report copies of values that own runtime state.
type noCopy struct{}

// Lock marks the pointer as a lock for the copylocks check; it does no runtime work.
func (*noCopy) Lock() {}

// Unlock completes the lock method set for the copylocks check.
func (*noCopy) Unlock() {}

// Service supplies the arguments saved by Start.
// Embed it by value. Start creates and initializes the service value.
// Service copies follow the Go copy and synchronization rules for A.
type Service[A any] struct {
	args A
}

// Args returns the arguments supplied to Start.
func (p *Service[A]) Args() A { return p.args }

// Init supplies empty setup when the service does not declare Init.
// The runtime supplies a supervisor for services started by this service.
// Keep its pointer for later starts. It stops when this service ends.
// A declared Init must return if a dependency stops answering. Bound blocking
// operations with an application timeout or an independently cancelled context.
// Init must release application resources if it fails. The runtime stops children
// created by that setup.
func (*Service[A]) Init(*Supervisor) error { return nil }

// bind saves the arguments before Init.
func (p *Service[A]) bind(args A) { p.args = args }

// NoArgs supplies empty start arguments.
type NoArgs struct{}

// Supervisor owns an ordered set of service lifetimes and nested supervisors.
// It activates at most once and stops at most once. The zero value is not usable.
// Do not copy an initialized Supervisor. Pass its pointer.
// Run go vet to check common copies; see the copy contract for its coverage limits.
type Supervisor struct {
	_         noCopy
	mu        *sync.Mutex
	children  childList
	starts    sync.WaitGroup
	active    bool
	stopping  bool
	stopped   chan struct{}
	admission context.Context
	cancel    context.CancelFunc
	root      *Supervisor
	parent    *Supervisor
	entry     *child
	report    func(any, []byte)
	exit      func(any)
	failure   *Failure
}

// childList preserves registration order and permits constant-time removal under Supervisor.mu.
type childList struct {
	first, last *child
	count       int
}

// child is pending while done is nil. Activation or shutdown starts its watcher.
// Nested supervisors own their own completion channel.
type child struct {
	run        func(context.Context)
	stop       func()
	done       chan struct{}
	group      *Supervisor
	prev, next *child
}

// Ref holds an initialized service value. Copies select the same value.
// Stop does not invalidate a reference or wait for direct service method calls.
type Ref[T any] struct {
	im.Frozen
	value im.UncheckedValue[T]
}

// IsZero reports whether this reference is unset.
func (p Ref[T]) IsZero() bool { return im.Unwrap(p.value) == nil }

// Borrow returns the initialized service value, including after Stop.
// The pointer can be stored and reused. Calls use ordinary Go synchronization.
// A zero reference panics with "supervise: zero Ref".
func (p Ref[T]) Borrow() *T {
	if im.Unwrap(p.value) == nil {
		panic("supervise: zero Ref")
	}
	return im.Unwrap(p.value)
}

// NewRoot creates an active lifetime supervisor. A Run panic or Goexit stops the root,
// waits for ordered cleanup, and panics with *Failure.
// report receives each recovered Run panic or Goexit once as *Failure, with its
// original stack, before that service waits for child cleanup. A nil report
// writes failures to standard error. Stop does not suppress failures.
// Reports can overlap, must return, and must not wait for their supervisor to stop.
func NewRoot(report func(any, []byte)) *Supervisor {
	return root(func(value any) { panic(value) }, report)
}

// root supplies the terminal exit hook used by runtime tests.
func root(exit func(any), report func(any, []byte)) *Supervisor {
	s := newSupervisor(nil, true)
	s.root, s.exit, s.report = s, exit, report
	return s
}

// newSupervisor creates a pending or active supervisor with its owner's admission context.
func newSupervisor(owner *Supervisor, active bool) *Supervisor {
	parent := context.Background()
	var mu *sync.Mutex
	var top *Supervisor
	if owner != nil {
		parent, top, mu = owner.admission, owner.root, owner.mu
	} else {
		mu = new(sync.Mutex)
	}
	admission, cancel := context.WithCancel(parent)
	return &Supervisor{
		mu: mu, active: active, stopped: make(chan struct{}), admission: admission, cancel: cancel, root: top,
	}
}

// StartSupervisor creates a lifetime supervisor for selective shutdown.
// A Run failure in any child stops the root.
// A stopped parent returns a stopped supervisor.
func StartSupervisor(parent *Supervisor) *Supervisor {
	checkSupervisor(parent)
	nested := newSupervisor(parent, false)
	nested.parent = parent
	parent.add(nil, nested)
	return nested
}

// Start creates T, calls Init once in the caller, and publishes the initialized value.
// An Init error returns a zero Ref and the error; an Init panic reaches
// the caller. The runtime stops staged children in both cases.
// A stopped parent returns a zero Ref and context.Canceled before Init.
// Each accepted, successful Init gets one Run call. Start returns after Init and
// registration; Run can receive a cancelled context if Stop closes admission.
// Stop waits for accepted Init calls before stopping registered services.
func Start[T, A any, P interface {
	*T
	Init(*Supervisor) error
	Run(context.Context)
	bind(A)
}](parent *Supervisor, args A) (Ref[T], error) {
	checkSupervisor(parent)
	parent.mu.Lock()
	err := parent.admission.Err()
	if err == nil {
		parent.starts.Add(1)
	}
	parent.mu.Unlock()
	if err != nil {
		return Ref[T]{}, err
	}
	defer parent.starts.Done()
	value := new(T)
	bindService(P(value), args)
	supervisor := newSupervisor(parent, false)
	ready := false
	defer func() {
		if !ready {
			<-supervisor.Stop()
		}
	}()
	if err := P(value).Init(supervisor); err != nil {
		return Ref[T]{}, err
	}
	ready = true
	ref := Ref[T]{value: im.Trust(value)}
	parent.add(func(ctx context.Context) {
		watch(parent, supervisor, ctx, reflect.TypeFor[T]().String(), P(value).Run)
	}, nil)
	return ref, nil
}

// bindService reports an invalid embedding path without catching application Init panics.
func bindService[A any, P interface {
	bind(A)
}](value P, args A) {
	defer func() {
		if recover() != nil {
			panic("supervise: embed Service by value through each embedded field")
		}
	}()
	value.bind(args)
}

// Stop closes admission in the subtree and requests ordered shutdown.
// It returns the same completion channel on every call. Shutdown waits for
// accepted Init calls, then stops children in reverse registration order and
// removes a nested supervisor from its parent. Init must return without waiting for
// shutdown of its owning supervisor or an ancestor, or cancellation that needs it.
// Stop does not wait for ordinary direct method calls. From Run, a service must
// not wait for a supervisor that contains that service. It can stop and wait for
// the supervisor passed to Init, which contains only the services it started.
func (s *Supervisor) Stop() <-chan struct{} {
	checkSupervisor(s)
	s.mu.Lock()
	defer s.mu.Unlock()
	s.stop()
	return s.stopped
}

// checkSupervisor rejects an absent supervisor before setup or shutdown can use it.
func checkSupervisor(s *Supervisor) {
	if s == nil || s.stopped == nil {
		panic("supervise: nil or zero Supervisor")
	}
}

// stop starts one shutdown worker. The caller holds s.mu.
func (s *Supervisor) stop() {
	if s.stopping {
		return
	}
	s.stopping = true
	s.cancel()
	go func() {
		s.starts.Wait()
		s.halt()
		s.mu.Lock()
		failure := s.failure
		if failure == nil {
			if s.parent != nil {
				s.parent.unlink(s.entry)
				s.parent, s.entry = nil, nil
			}
			close(s.stopped)
			s.mu.Unlock()
			return
		}
		s.mu.Unlock()
		s.exit(failure)
		close(s.stopped)
	}()
}

// fail records the first failure before reporting or cleanup can run.
func (s *Supervisor) fail(failure *Failure) {
	top := s.root
	top.mu.Lock()
	defer top.mu.Unlock()
	if top.failure != nil {
		return
	}
	top.failure = failure
	top.stop()
}

// add registers an accepted service after Init, or admits a new nested supervisor.
func (s *Supervisor) add(run func(context.Context), group *Supervisor) {
	c := &child{run: run, group: group}
	if group != nil {
		group.entry = c
	}
	s.mu.Lock()
	if group != nil && s.admission.Err() != nil {
		s.mu.Unlock()
		<-group.Stop()
		return
	}
	defer s.mu.Unlock()
	c.prev = s.children.last
	if c.prev != nil {
		c.prev.next = c
	} else {
		s.children.first = c
	}
	s.children.last = c
	s.children.count++
	if s.active {
		s.launch(c)
	}
}

// activate launches pending entries once. The caller holds the tree lock.
// Parent shutdown keeps them pending for cleanup.
func (s *Supervisor) activate() {
	if s.active || s.admission.Err() != nil {
		return
	}
	s.active = true
	for c := s.children.first; c != nil; c = c.next {
		s.launch(c)
	}
}

// finish publishes completion and removal in one parent-list operation.
func (s *Supervisor) finish(c *child, done chan struct{}) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.unlink(c)
	close(done)
}

// unlink removes an entry in constant time. The caller holds s.mu.
func (s *Supervisor) unlink(c *child) {
	c.run = nil
	if c.prev == nil && s.children.first != c {
		return
	}
	if c.prev != nil {
		c.prev.next = c.next
	} else {
		s.children.first = c.next
	}
	if c.next != nil {
		c.next.prev = c.prev
	} else {
		s.children.last = c.prev
	}
	s.children.count--
	c.prev, c.next = nil, nil
}

// launch gives a service's pending value to one watcher. The caller holds s.mu.
func (s *Supervisor) launch(c *child) {
	if c.group != nil {
		c.group.activate()
		return
	}
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	c.stop, c.done = cancel, done
	run := c.run
	go func() {
		defer s.finish(c, done)
		defer cancel()
		run(ctx)
	}()
}

// halt removes the last child after it stops. Admission is already closed.
func (s *Supervisor) halt() {
	for {
		s.mu.Lock()
		c := s.children.last
		if c == nil {
			s.mu.Unlock()
			return
		}
		if c.group == nil && c.done == nil {
			s.launch(c)
		}
		stop, done := c.stop, c.done
		s.mu.Unlock()
		if c.group != nil {
			<-c.group.Stop()
			continue
		}
		stop()
		<-done
	}
}

// watch owns Run cancellation and reports a failure before it waits for child cleanup.
func watch(parent, supervisor *Supervisor, ctx context.Context, name string, run func(context.Context)) {
	runContext, cancel := context.WithCancel(context.WithoutCancel(ctx))
	var unwatch func() bool
	defer func() {
		if unwatch != nil {
			unwatch()
		}
		cancel()
		<-supervisor.Stop()
	}()
	completed := false
	defer func() {
		if completed {
			return
		}
		value := recover()
		if value == nil {
			value = errGoexit
		}
		stack := debug.Stack()
		failure := &Failure{Service: name, Value: value, Stack: stack}
		parent.fail(failure)
		parent.root.reportFailure(failure)
	}()
	halt := func() { <-supervisor.Stop(); cancel() }
	parent.mu.Lock()
	admitted := parent.admission.Err() == nil && ctx.Err() == nil
	if admitted {
		unwatch = context.AfterFunc(ctx, halt)
		supervisor.activate()
	}
	parent.mu.Unlock()
	if !admitted {
		<-ctx.Done()
		halt()
	}
	run(runContext)
	completed = true
}
