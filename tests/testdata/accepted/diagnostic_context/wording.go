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

func convert(a A) B { return B{N: a.N} }

type ArgumentsWorker struct{ supervise.Service[Score] }

func (*ArgumentsWorker) Run(context.Context) {}

func (w *Worker) Handle(ref supervise.Ref[Worker]) supervise.Ref[Worker] { return ref }
