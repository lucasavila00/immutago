package service

import (
	"context"

	"github.com/lucasavila00/immutago/pkg/supervise"
)

// Buffer is a service with rejected receiver uses.
type Buffer struct {
	supervise.Service[supervise.NoArgs]
	size int
}

// Run waits until ctx ends.
func (b *Buffer) Run(ctx context.Context) { <-ctx.Done() }

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
	*b = Buffer{}
	marker := b.Service
	b.Service = supervise.Service[supervise.NoArgs]{}
	args := b.Args
	_, _, _, _, _ = other, later, grow, marker, args
}
