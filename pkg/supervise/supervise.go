// Package supervise runs services with independent retries and owned child lifetimes.
//
// Embed Service[A], supply Run(context.Context), and optionally declare Init() error.
// Start initializes the service and returns Ref[T]. Calls through Borrow run in
// the caller's goroutine; review shared fields, locks, and inputs in service code.
// Groups control lifetime and stop order. They do not restart services together.
package supervise

import (
	"context"
	"errors"
	"reflect"
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

var errGoexit = errors.New("supervise: service called runtime.Goexit")

// Service supplies start arguments and the children owned by one service attempt.
// Embed it by value. Start creates and initializes each fresh service value.
type Service[A any] struct {
	args     A
	owner    *Group
	mu       sync.Mutex
	cancel   context.CancelFunc
	unwatch  func() bool
	children *Group
	ended    bool
}

// Args returns the arguments supplied to Start. Every retry receives the same arguments.
func (p *Service[A]) Args() A { return p.args }

// Init supplies empty setup when the service does not declare Init.
// A declared Init must release application resources if it fails. The runtime
// stops children created by that setup attempt.
func (*Service[A]) Init() error { return nil }

// Children returns the group owned by this service attempt. Its services wait
// until the parent Run starts, or run with cancelled contexts during cleanup.
// The runtime stops and waits for these children
// before it replaces the parent. A later attempt owns a new group.
// After this attempt ends, Children returns a stopped group.
func (p *Service[A]) Children() *Group {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.children == nil {
		p.children = newGroup(p.owner, false)
		switch {
		case p.ended:
			p.children.Stop()
		case p.cancel != nil:
			p.children.activate()
		}
	}
	return p.children
}

// bind supplies arguments and lifetime ownership before Init.
func (p *Service[A]) bind(args A, owner *Group) {
	p.args, p.owner = args, owner
}

// begin activates children and connects ordered shutdown to this Run context.
func (p *Service[A]) begin(ctx context.Context) context.Context {
	p.mu.Lock()
	run, cancel := context.WithCancel(context.WithoutCancel(ctx))
	p.cancel = cancel
	if ctx.Err() != nil {
		p.ended = true
		p.mu.Unlock()
		p.halt()
		return run
	}
	p.unwatch = context.AfterFunc(ctx, p.halt)
	if p.children != nil {
		p.children.activate()
	}
	p.mu.Unlock()
	return run
}

// halt stops children before it cancels the parent's Run context.
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

// end closes child admission and waits for cleanup after Init or Run ends.
func (p *Service[A]) end() {
	if p.unwatch != nil {
		p.unwatch()
	}
	p.mu.Lock()
	p.ended = true
	if p.cancel != nil {
		p.cancel()
	}
	children := p.children
	p.mu.Unlock()
	if children != nil {
		<-children.Stop()
	}
}

// NoArgs supplies empty start arguments.
type NoArgs struct{}

// Group owns an ordered set of service lifetimes and nested groups.
// It activates at most once and stops at most once. The zero value is not usable.
type Group struct {
	mu       sync.Mutex
	children childList
	active   bool
	stopping bool
	stopped  chan struct{}
	policy   context.Context
	cancel   context.CancelFunc
	root     *Group
	parent   *Group
	entry    *child
	report   func(any, []byte)
	exit     func(any)
}

// childList preserves start order and permits constant-time removal under Group.mu.
type childList struct {
	first, last *child
	count       int
}

// child is pending while done is nil. Activation or shutdown starts its watcher.
// Nested groups own their own completion channel.
type child struct {
	run        func(context.Context)
	stop       func()
	done       chan struct{}
	group      *Group
	prev, next *child
}

// Ref selects the latest initialized service value. Copies share one atomic slot.
// Stop does not invalidate a reference or wait for direct service method calls.
type Ref[T any] struct{ value *atomic.Pointer[T] }

// IsZero reports whether this reference is unset.
func (p Ref[T]) IsZero() bool { return p.value == nil }

// Borrow selects the latest initialized value with one atomic load.
// Calls can finish on an older value after a retry. Borrow is shared, not exclusive.
// A zero reference panics with "supervise: zero Ref".
func (p Ref[T]) Borrow() *T {
	if p.value == nil {
		panic("supervise: zero Ref")
	}
	return p.value.Load()
}

// NewRoot creates an active lifetime group. An exhausted service or Goexit stops
// the root, waits for ordered cleanup, and panics with *Failure.
// report receives each recovered panic, restart Init error, or Goexit with its original
// stack, then the selected terminal *Failure before cleanup waits. A nil report
// disables attempt reports; terminal failures still go to standard error.
// Reports can overlap, must return, and must not wait for their group to stop.
func NewRoot(report func(any, []byte)) *Group {
	return root(func(value any) { panic(value) }, report)
}

// root supplies the terminal exit hook used by runtime tests.
func root(exit func(any), report func(any, []byte)) *Group {
	s := newGroup(nil, true)
	s.root, s.exit, s.report = s, exit, report
	return s
}

// newGroup creates a pending or active group with its owner's retry policy.
func newGroup(owner *Group, active bool) *Group {
	parent := context.Background()
	var top *Group
	if owner != nil {
		parent, top = owner.policy, owner.root
	}
	policy, cancel := context.WithCancel(parent)
	return &Group{active: active, stopped: make(chan struct{}), policy: policy, cancel: cancel, root: top}
}

// StartGroup creates a lifetime group for selective shutdown. It does not add a
// restart budget. An exhausted failure goes directly to the root.
// A stopped parent returns a stopped group.
func StartGroup(parent *Group) *Group {
	checkGroup(parent)
	nested := newGroup(parent, false)
	nested.parent = parent
	parent.add(nil, nested)
	return nested
}

// Start creates T, calls Init in the caller, and publishes the initialized value.
// A first Init error returns a zero Ref and the error; a first Init panic reaches
// the caller. The runtime stops staged children in both cases.
// Each successful Init gets one Run call, with a cancelled context if its attempt
// cannot activate. Start under a stopped parent waits for that cleanup Run.
// Stop waits for registered children, not an initial Init that has not registered yet.
func Start[T, A any, P interface {
	*T
	Init() error
	Run(context.Context)
	bind(A, *Group)
	begin(context.Context) context.Context
	end()
}](parent *Group, args A) (Ref[T], error) {
	checkGroup(parent)
	create := func() *T {
		value := new(T)
		bindService(P(value), args, parent)
		return value
	}
	first := create()
	ready := false
	defer func() {
		if !ready {
			P(first).end()
		}
	}()
	if err := P(first).Init(); err != nil {
		return Ref[T]{}, err
	}
	ready = true
	ref := Ref[T]{value: new(atomic.Pointer[T])}
	ref.value.Store(first)
	current := first
	run := func(ctx context.Context) {
		value := current
		if value == nil {
			if parent.policy.Err() != nil || ctx.Err() != nil {
				return
			}
			value = create()
			current = value
			if err := P(value).Init(); err != nil {
				panic(err)
			}
		}
		service := P(value)
		if parent.policy.Err() != nil {
			ctx = parent.policy
		}
		if parent.policy.Err() == nil && ctx.Err() == nil {
			ref.value.Store(value)
		}
		service.Run(service.begin(ctx))
	}
	cleanup := func() {
		if current != nil {
			P(current).end()
			current = nil
		}
	}
	parent.add(func(ctx context.Context) {
		watch(parent, ctx, reflect.TypeFor[T]().String(), run, cleanup)
	}, nil)
	return ref, nil
}

// bindService reports an invalid embedding path without catching application Init panics.
func bindService[A any, P interface{ bind(A, *Group) }](value P, args A, parent *Group) {
	defer func() {
		if recover() != nil {
			panic("supervise: embed Service by value through each embedded field")
		}
	}()
	value.bind(args, parent)
}

// Stop prevents new retries in the subtree and requests ordered shutdown.
// It returns the same completion channel on every call. Shutdown waits for
// children in reverse start order, then removes a nested group from its parent.
// It does not wait for ordinary direct method calls. A service can request its
// group's stop, but must not wait for that group from its own Run.
func (s *Group) Stop() <-chan struct{} {
	checkGroup(s)
	s.mu.Lock()
	defer s.mu.Unlock()
	s.stop(nil)
	return s.stopped
}

// checkGroup rejects an absent group before setup or shutdown can use it.
func checkGroup(s *Group) {
	if s == nil || s.stopped == nil {
		panic("supervise: nil or zero Group")
	}
}

// stop starts one shutdown worker. The caller holds s.mu.
func (s *Group) stop(failure *Failure) {
	if s.stopping {
		return
	}
	s.stopping = true
	s.cancel()
	go func() {
		if failure != nil {
			s.reportFailure(failure)
		}
		s.halt()
		if failure != nil {
			s.exit(failure)
		}
		if s.parent != nil {
			parent, entry := s.parent, s.entry
			s.parent, s.entry = nil, nil
			parent.finish(entry, s.stopped)
		} else {
			close(s.stopped)
		}
	}()
}

// fail submits one terminal failure. Root Stop and failure use the same decision lock.
func (s *Group) fail(failure *Failure) {
	top := s.root
	top.mu.Lock()
	defer top.mu.Unlock()
	if s.policy.Err() == nil {
		top.stop(failure)
	}
}

// add registers pending setup or starts one lifetime. Initial Init is already complete.
func (s *Group) add(run func(context.Context), group *Group) {
	c := &child{run: run, group: group}
	if group != nil {
		group.entry = c
	}
	s.mu.Lock()
	if s.stopping || s.policy.Err() != nil {
		if group != nil {
			s.mu.Unlock()
			<-group.Stop()
		} else {
			s.launch(c)
			done := c.done
			s.mu.Unlock()
			<-done
		}
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

// activate launches pending entries once. Parent shutdown keeps them pending for cleanup.
func (s *Group) activate() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.active || s.stopping || s.policy.Err() != nil {
		return
	}
	s.active = true
	for c := s.children.first; c != nil; c = c.next {
		s.launch(c)
	}
}

// forget removes an entry whose work has already ended.
func (s *Group) forget(c *child) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.unlink(c)
}

// finish publishes completion and removal in one parent-list operation.
func (s *Group) finish(c *child, done chan struct{}) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.unlink(c)
	close(done)
}

// unlink removes an entry in constant time. The caller holds s.mu.
func (s *Group) unlink(c *child) {
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
func (s *Group) launch(c *child) {
	if c.group != nil {
		c.group.activate()
		return
	}
	ctx, cancel := context.WithCancel(context.Background())
	if s.stopping || s.policy.Err() != nil {
		cancel()
	}
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
func (s *Group) halt() {
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
		if done != nil {
			<-done
		}
		s.forget(c)
	}
}

// watch owns one service's retry budget. Every return ends that service entry.
func watch(parent *Group, ctx context.Context, name string, run func(context.Context), cleanup func()) {
	report := parent.root.report
	var restarts []time.Time
	var failureAt time.Time
	terminal := false
	failed := func(value any, stack []byte, goexit bool) {
		failureAt = time.Now()
		if goexit {
			value = errGoexit
		}
		if report != nil {
			report(value, stack)
		}
		if parent.policy.Err() != nil || ctx.Err() != nil {
			return
		}
		restarts = slices.DeleteFunc(restarts, func(at time.Time) bool { return failureAt.Sub(at) >= restartWindow })
		if goexit || len(restarts) == restartLimit {
			terminal = true
			parent.fail(&Failure{Service: name, Value: value, Stack: stack})
		}
	}
	for {
		panicked := attempt(ctx, run, cleanup, report != nil || len(restarts) == restartLimit, failed)
		if !panicked || terminal || parent.policy.Err() != nil || ctx.Err() != nil {
			return
		}
		delay := time.NewTimer(time.Second << len(restarts))
		restarts = append(restarts, failureAt)
		select {
		case <-delay.C:
		case <-parent.policy.Done():
			delay.Stop()
			return
		case <-ctx.Done():
			delay.Stop()
			return
		}
	}
}

// attempt reports panic or Goexit before cleanup can wait for children.
func attempt(ctx context.Context, run func(context.Context), cleanup func(), captureStack bool,
	report func(any, []byte, bool),
) (failed bool) {
	defer cleanup()
	defer func() {
		if failed {
			value := recover()
			var stack []byte
			if captureStack || value == nil {
				stack = debug.Stack()
			}
			report(value, stack, value == nil)
		}
	}()
	failed = true
	run(ctx)
	return false
}
