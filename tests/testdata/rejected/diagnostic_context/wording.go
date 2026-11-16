package diagnosticcontext

import (
	"context"

	"github.com/lucasavila00/immutago/pkg/im"
	"github.com/lucasavila00/immutago/pkg/supervise"
)

type A struct {
	im.Local
	N int
}

type B A

func convert(a A) B { return B(a) }

type ArgumentsWorker struct{ supervise.Service[A] }

func (*ArgumentsWorker) Run(context.Context) {}

func (w *Worker) Self() *Worker { return w }
