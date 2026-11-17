package accepted

import (
	"context"

	"github.com/lucasavila00/immutago/pkg/supervise"
)

type Worker struct {
	supervise.Service[supervise.NoArgs]
	supervisor *supervise.Supervisor
}

func (w *Worker) Init(supervisor *supervise.Supervisor) error { w.supervisor = supervisor; return nil }

func (*Worker) Run(ctx context.Context) { <-ctx.Done() }

func (w *Worker) AddChild() bool {
	_, err := supervise.Start[Worker](w.supervisor, supervise.NoArgs{})
	return err == nil
}

type Holder struct {
	Worker     *Worker
	Supervisor *supervise.Supervisor
	Ref        supervise.Ref[Worker]
}

func Share(worker *Worker, group *supervise.Supervisor, ref supervise.Ref[Worker]) Holder {
	other := ref
	return Holder{Worker: worker, Supervisor: group, Ref: other}
}

func Start(group *supervise.Supervisor) (supervise.Ref[Worker], error) {
	parent := group
	return supervise.Start[Worker](parent, supervise.NoArgs{})
}
