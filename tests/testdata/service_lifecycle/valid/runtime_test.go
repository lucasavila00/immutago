package valid

import (
	"context"
	"sync"
	"testing"

	"github.com/lucasavila00/immutago/pkg/supervise"
)

type Counter struct {
	supervise.Service[chan int]
	mu    *sync.Mutex
	limit int
}

func (c *Counter) Init(*supervise.Supervisor) error {
	c.mu = new(sync.Mutex)
	c.limit = 5
	return nil
}

func (c *Counter) SetLimit(n int) { c.mu.Lock(); c.limit = n; c.mu.Unlock() }

func (c *Counter) Run(context.Context) {
	c.mu.Lock()
	n := c.limit
	c.mu.Unlock()
	c.Args() <- n
}

func TestLockedPointerRun(t *testing.T) {
	for range 50 {
		root := supervise.NewRoot(nil)
		result := make(chan int, 1)
		ref, err := supervise.Start[Counter](root, result)
		if err != nil {
			t.Fatal(err)
		}
		if ref.Borrow().limit != 5 {
			t.Fatal("Init did not set the original service")
		}
		for n := range 1000 {
			ref.Borrow().SetLimit(n)
		}
		<-result
		<-root.Stop()
	}
}

func TestLifecycleForms(t *testing.T) {
	root := supervise.NewRoot(nil)
	StartAll(root)
	<-root.Stop()
}
