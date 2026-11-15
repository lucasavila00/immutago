package shadow

import (
	"context"

	"github.com/lucasavila00/immutago/pkg/supervise"
)

type Base struct {
	supervise.Service[supervise.NoArgs]
}
type Middle struct{ Base }
type Worker struct {
	Middle
	*supervise.Service[supervise.NoArgs]
}

func (*Worker) Run(ctx context.Context) { <-ctx.Done() }

var ref supervise.Ref[Worker]
