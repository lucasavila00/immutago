package supervise_test

import (
	"context"
	"testing"
	"testing/synctest"

	"github.com/lucasavila00/immutago/pkg/supervise"
)

// copyLifecycle supplies channel boundaries for an owner and its child.
type copyLifecycle struct {
	parentDone   chan struct{}
	childStarted chan struct{}
	childStopped chan struct{}
}

// copyParent saves the supervisor in an ordinary application field.
type copyParent struct {
	supervise.Service[*copyLifecycle]
	supervisor *supervise.Supervisor
}

// Init saves the supervisor before the service can be copied.
func (p *copyParent) Init(supervisor *supervise.Supervisor) error {
	p.supervisor = supervisor
	return nil
}

// Run ends on shutdown or when the test requests normal completion.
func (p *copyParent) Run(ctx context.Context) {
	select {
	case <-ctx.Done():
	case <-p.Args().parentDone:
	}
}

// AddChild uses the same owned supervisor from a copied receiver.
func (p copyParent) AddChild() bool {
	_, err := supervise.Start[copyChild](p.supervisor, p.Args())
	return err == nil
}

// copyChild reports entry and completion through channels.
type copyChild struct {
	supervise.Service[*copyLifecycle]
}

// Run confirms that the child has started before its owner ends.
func (p *copyChild) Run(ctx context.Context) {
	close(p.Args().childStarted)
	<-ctx.Done()
	close(p.Args().childStopped)
}

// TestServiceCopiesShareSupervisor keeps starts from a copy in the original owner's lifetime.
func TestServiceCopiesShareSupervisor(t *testing.T) {
	for _, mode := range []string{"return", "stop"} {
		t.Run(mode, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) { checkServiceCopySupervisor(t, mode) })
		})
	}
}

// checkServiceCopySupervisor ends one owner after a copy starts its child.
func checkServiceCopySupervisor(t *testing.T, mode string) {
	t.Helper()
	root := supervise.NewRoot(nil)
	defer func() { <-root.Stop() }()
	args := &copyLifecycle{make(chan struct{}), make(chan struct{}), make(chan struct{})}
	ref, err := supervise.Start[copyParent](root, args)
	if err != nil {
		t.Fatal(err)
	}
	copy := *ref.Borrow()
	if copy.supervisor != ref.Borrow().supervisor || !copy.AddChild() {
		t.Fatal("copy did not start a child under the same supervisor")
	}
	<-args.childStarted
	if mode == "return" {
		close(args.parentDone)
	} else {
		root.Stop()
	}
	synctest.Wait()
	select {
	case <-args.childStopped:
	default:
		t.Fatal("child is still running after its owner ended")
	}
	if copy.AddChild() || ref.Borrow().AddChild() {
		t.Fatal("a late start was accepted after the owner ended")
	}
}
