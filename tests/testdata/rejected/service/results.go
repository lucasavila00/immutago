package service

import (
	"context"
	"github.com/lucasavila00/immutago/pkg/im"
	"github.com/lucasavila00/immutago/pkg/supervise"
)

type Reply struct {
	im.Scoped
	Count int
}
type SharedReply struct {
	im.Shared
	Count int
}
type Counter struct {
	supervise.Service[supervise.NoArgs]
}

func (*Counter) Run(context.Context)           {}
func (*Counter) Pair() (int, bool)             { return 1, true }
func (*Counter) Map() map[string]int           { return nil }
func (*Counter) Slice() []int                  { return nil }
func (*Counter) Interface() error              { return nil }
func (*Counter) Pointer() *SharedReply         { return &SharedReply{} }
func (*Counter) ScopedPointer() *Reply         { return nil }
func (*Counter) Write(p *Reply)                { p.Count++ }
func (*Counter) Capture(p *Reply)              { f := func() int { return p.Count }; _ = f() }
func (*Counter) Frozen() im.FrozenSlice[[]int] { var result im.FrozenSlice[[]int]; return result }

type Unmarked struct{ Count int }

func (*Counter) Struct() Unmarked { return Unmarked{} }

type Methods struct{}

func (Methods) Mutable() []int { return nil }

type Promoted struct {
	Counter
	Methods
}
type PointerEmbedded struct{ *supervise.Service[int] }

func (*PointerEmbedded) Mutable() []int { return nil }

// The runtime stores arguments, so a Scoped pointer cannot be a start argument.
type StoredPointer struct{ supervise.Service[*Reply] }

func (*Counter) Shared() SharedReply           { return SharedReply{} }
func (*Counter) SharedArray() [1]SharedReply   { return [1]SharedReply{} }
func (*Counter) PointerArray() [1]*SharedReply { return [1]*SharedReply{} }
func (*Counter) List() im.FrozenSlice[int]     { var value im.FrozenSlice[int]; return value }
func (*Counter) MapResult() im.FrozenMap[string, int] {
	var value im.FrozenMap[string, int]
	return value
}
