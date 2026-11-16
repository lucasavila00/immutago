package supervise

import (
	"context"
	"testing"
	"testing/synctest"
)

// copyLifecycle supplies channel boundaries for a method that creates the first child group.
type copyLifecycle struct {
	parentStarted chan struct{}
	childStarted  chan struct{}
	childStopped  chan struct{}
	groups        chan *Group
}

// copyParent creates children through a pointer receiver after Run starts.
type copyParent struct{ Service[*copyLifecycle] }

// Run reports admission before the test calls AddChild.
func (p *copyParent) Run(ctx context.Context) { close(p.Args().parentStarted); <-ctx.Done() }

// AddChild keeps the first child group on the service value owned by the runtime.
func (p *copyParent) AddChild() bool {
	group := p.UnsafeChildren()
	p.Args().groups <- group
	_, err := Start[copyChild](group, p.Args())
	return err == nil
}

// copyChild reports entry and completion through channels.
type copyChild struct{ Service[*copyLifecycle] }

// Run confirms that the child has started before the test stops the root.
func (p *copyChild) Run(ctx context.Context) {
	close(p.Args().childStarted)
	<-ctx.Done()
	close(p.Args().childStopped)
}

// TestPointerReceiverOwnsChildren checks the pointer replacement for a copied receiver.
func TestPointerReceiverOwnsChildren(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		root := NewRoot(nil)
		defer func() { <-root.Stop() }()
		args := &copyLifecycle{
			parentStarted: make(chan struct{}),
			childStarted:  make(chan struct{}),
			childStopped:  make(chan struct{}),
			groups:        make(chan *Group, 1),
		}
		ref, err := Start[copyParent](root, args)
		if err != nil {
			t.Fatal(err)
		}
		<-args.parentStarted
		other := ref
		if !other.Borrow().AddChild() {
			t.Fatal("child setup failed")
		}
		group := <-args.groups
		defer func() { <-group.Stop() }()
		<-args.childStarted
		<-root.Stop()
		select {
		case <-args.childStopped:
		default:
			t.Fatal("child is still running after root Stop")
		}
	})
}
