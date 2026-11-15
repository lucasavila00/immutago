package main

import (
	"context"
	"fmt"

	"github.com/lucasavila00/immutago/pkg/supervise"
)

// Table has one slot for each entry requested at startup.
type Table struct {
	supervise.Service[int]
	slots []int
}

// Init prepares the slots before callers can read them.
func (t *Table) Init() error {
	if t.Args() < 1 {
		return fmt.Errorf("slot count must be positive")
	}
	t.slots = make([]int, t.Args())
	t.slots[0] = 42
	return nil
}

// Run waits for shutdown.
func (*Table) Run(ctx context.Context) { <-ctx.Done() }

// First reads the first slot, including before Run begins.
func (t *Table) First() int { return t.slots[0] }

// main checks successful setup, a returned failure, and a new attempt after failure.
func main() {
	root := supervise.NewRoot(nil)
	table, err := supervise.Start[Table](root, 3)
	if err != nil {
		panic(err)
	}
	fmt.Println(table.Borrow().First())
	failed, err := supervise.Start[Table](root, 0)
	fmt.Println("unset:", failed.IsZero())
	fmt.Println(err)
	var zero supervise.Ref[Table]
	fmt.Println("zero:", zero.IsZero())
	retry, err := supervise.Start[Table](root, 1)
	if err != nil {
		panic(err)
	}
	if !retry.IsZero() {
		fmt.Println(retry.Borrow().First())
	}
	<-root.Stop()
	fmt.Println("stopped:", retry.IsZero())
}

// startFailure forwards the startup results without a result binding.
func startFailure(root *supervise.Group) (supervise.Ref[Table], error) {
	return supervise.Start[Table](root, 0)
}
