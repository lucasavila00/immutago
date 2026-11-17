package protected

import (
	"context"

	"example.com/consumer/hooks"
	"github.com/lucasavila00/immutago/pkg/im"
	"github.com/lucasavila00/immutago/pkg/supervise"
)

type Score struct {
	im.Copied
	N int
}

type Worker struct {
	supervise.Service[Score]
	hooks.ValueRun
}

func (*Worker) Run(context.Context) {}

type Rejected Worker

func (Rejected) Run(context.Context) {}

type Interface struct {
	supervise.Service[Score]
	hooks.Runner
}
