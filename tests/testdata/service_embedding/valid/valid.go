package valid

import (
	"context"

	"github.com/lucasavila00/immutago/pkg/supervise"
)

type Marker = supervise.Service[supervise.NoArgs]
type Direct struct{ Marker }

func (*Direct) Run(ctx context.Context) { <-ctx.Done() }
func (*Direct) Count() int              { return 1 }

type Base = Direct
type Middle struct{ Base }
type State struct{ Count int }
type Nested struct {
	Middle
	*State
}

func (*Nested) Run(ctx context.Context) { <-ctx.Done() }

type NestedAlias = Nested

type Defined Direct

func (*Defined) Run(ctx context.Context) { <-ctx.Done() }

type Recursive struct {
	Marker
	*Recursive
}

func (*Recursive) Run(ctx context.Context) { <-ctx.Done() }

type Anonymous = struct{ Direct }

var defined supervise.Ref[Defined]
var recursive supervise.Ref[Recursive]
var anonymous supervise.Ref[Anonymous]

var direct supervise.Ref[Direct]
var nested supervise.Ref[NestedAlias]

func start(root *supervise.Supervisor) error {
	var err error
	direct, err = supervise.Start[Direct](root, supervise.NoArgs{})
	if err != nil {
		return err
	}
	_, err = supervise.Start[NestedAlias](root, supervise.NoArgs{})
	return err
}

func read() int { return direct.Borrow().Count() }
