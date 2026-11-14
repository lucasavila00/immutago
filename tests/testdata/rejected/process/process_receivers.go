package process

import (
	"context"

	"github.com/lucasavila00/immutago/pkg/proc"
)

// Buffer is a process with rejected receiver uses.
type Buffer struct {
	proc.Process[proc.None]
	size int
}

// Run waits until ctx ends.
func (b *Buffer) Run(ctx context.Context) { <-ctx.Done() }

// Init sets the first size.
func (b *Buffer) Init() { b.size = 1 }

// grow increases the size.
func (b *Buffer) grow() { b.size++ }

// misuse uses the receiver in each rejected form.
func (b *Buffer) misuse(ctx context.Context) {
	other := b
	use(b)
	later := func() { b.size++ }
	go b.grow()
	defer b.grow()
	grow := b.grow
	b.Run(ctx)
	b.Init()
	*b = Buffer{}
	marker := b.Process
	b.Process = proc.Process[proc.None]{}
	args := b.Args
	_, _, _, _, _ = other, later, grow, marker, args
}
