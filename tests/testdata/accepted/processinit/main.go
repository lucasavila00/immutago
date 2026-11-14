package main

import (
	"context"
	"fmt"

	"github.com/lucasavila00/immutago/pkg/proc"
)

// Table has one slot for each entry requested at startup.
type Table struct {
	proc.Process[int]
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

// main checks successful setup, a returned failure, and a retry on a reserved PID.
func main() {
	root := proc.NewRoot(nil)
	table, err := proc.Start[Table](root, 3)
	if err != nil {
		panic(err)
	}
	fmt.Println(table.Proc().First())
	failed, err := proc.Start[Table](root, 0)
	_ = failed
	fmt.Println(err)
	reserved := proc.Reserve[Table]()
	err = proc.StartReserved(root, reserved, 0)
	fmt.Println(err)
	if err := proc.StartReserved(root, reserved, 1); err != nil {
		panic(err)
	}
	fmt.Println(reserved.Proc().First())
	<-root.Stop()
}

// startFailure forwards the startup results without a result binding.
func startFailure(root *proc.Supervisor) (proc.PID[Table], error) {
	return proc.Start[Table](root, 0)
}
