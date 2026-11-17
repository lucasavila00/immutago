package hooks

import (
	"context"

	"github.com/lucasavila00/immutago/pkg/supervise"
)

// These ordinary types can have value methods with lifecycle names.
type ValueRun struct{}

func (ValueRun) Run(context.Context) {}

type ValueInit struct{}

func (ValueInit) Init(*supervise.Supervisor) error { return nil }

type PointerRun struct{}

func (*PointerRun) Run(context.Context) {}

type PointerInit struct{}

func (*PointerInit) Init(*supervise.Supervisor) error { return nil }

type Runner interface{ Run(context.Context) }
type Initializer interface {
	Init(*supervise.Supervisor) error
}
