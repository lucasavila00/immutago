package nested

import (
	"context"

	"github.com/lucasavila00/immutago/pkg/supervise"
)

type Base struct {
	supervise.Service[supervise.NoArgs]
}
type BaseAlias = Base
type Pointer = *BaseAlias
type Worker struct{ Pointer }

func (*Worker) Run(ctx context.Context) { <-ctx.Done() }

var ref supervise.Ref[Worker]
