package valid

import (
	"context"

	"example.com/consumer/hooks"
	"github.com/lucasavila00/immutago/pkg/supervise"
)

type Direct struct{ supervise.Service[int] }

func (*Direct) Init(*supervise.Supervisor) error { return nil }
func (*Direct) Run(context.Context)              {}
func (w Direct) Count() int                      { return w.Args() }

type Alias = Direct
type Defined Direct

func (*Defined) Run(context.Context) {}

type AliasMethod struct{ supervise.Service[int] }
type PointerReceiver = *AliasMethod

func (PointerReceiver) Run(context.Context) {}

type Generic[A any] struct{ supervise.Service[A] }

func (*Generic[A]) Run(context.Context) {}

type GenericAlias[A any] = Generic[A]
type GenericPromoted[A any] struct{ Generic[A] }
type GenericDefined[A any] Generic[A]

func (*GenericDefined[A]) Run(context.Context) {}

type Promoted struct{ Direct }
type Anonymous = struct{ Direct }
type _ struct{ Direct }
type _ = Direct

type PromotedValue struct {
	supervise.Service[int]
	hooks.PointerRun
}

type PromotedPointer struct {
	supervise.Service[int]
	*hooks.PointerRun
}

func (w *PromotedPointer) Init(*supervise.Supervisor) error {
	w.PointerRun = &hooks.PointerRun{}
	return nil
}

type Base struct{ supervise.Service[int] }
type PromotedInit struct {
	Base
	hooks.PointerInit
}

func (*PromotedInit) Run(context.Context) {}

type PromotedPointerInit struct {
	Base
	*hooks.PointerInit
}

func (*PromotedPointerInit) Run(context.Context) {}

type Override struct {
	supervise.Service[int]
	hooks.ValueRun
}

func (*Override) Run(context.Context) {}

type OverrideInit struct {
	Base
	hooks.ValueInit
}

func (*OverrideInit) Init(*supervise.Supervisor) error { return nil }
func (*OverrideInit) Run(context.Context)              {}

type Parenthesized (struct {
	supervise.Service[int]
	hooks.ValueRun
})

func (*Parenthesized) Run(context.Context) {}

type Forward struct {
	supervise.Service[int]
	hooks.Runner
}

func (w *Forward) Init(*supervise.Supervisor) error { w.Runner = hooks.ValueRun{}; return nil }
func (w *Forward) Run(ctx context.Context)          { w.Runner.Run(ctx) }

func StartGeneric[A any](root *supervise.Supervisor, args A) (supervise.Ref[GenericAlias[A]], error) {
	return supervise.Start[GenericAlias[A]](root, args)
}

func StartAll(root *supervise.Supervisor) {
	_, _ = supervise.Start[Alias](root, 1)
	_, _ = supervise.Start[Defined](root, 1)
	_, _ = supervise.Start[AliasMethod](root, 1)
	_, _ = StartGeneric(root, 1)
	_, _ = supervise.Start[GenericPromoted[int]](root, 1)
	_, _ = supervise.Start[GenericDefined[int]](root, 1)
	_, _ = supervise.Start[Promoted](root, 1)
	_, _ = supervise.Start[Anonymous](root, 1)
	_, _ = supervise.Start[PromotedValue](root, 1)
	_, _ = supervise.Start[PromotedPointer](root, 1)
	_, _ = supervise.Start[PromotedInit](root, 1)
	_, _ = supervise.Start[PromotedPointerInit](root, 1)
	_, _ = supervise.Start[Override](root, 1)
	_, _ = supervise.Start[OverrideInit](root, 1)
	_, _ = supervise.Start[Parenthesized](root, 1)
	_, _ = supervise.Start[Forward](root, 1)
	_, _ = supervise.Start[struct{ Direct }](root, 1)
	type Local struct{ Direct }
	_, _ = supervise.Start[Local](root, 1)
}
