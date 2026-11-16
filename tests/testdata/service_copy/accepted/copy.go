package accepted

import (
	"context"

	"github.com/lucasavila00/immutago/pkg/supervise"
)

type Worker struct {
	supervise.Service[supervise.NoArgs]
}

func (*Worker) Run(ctx context.Context) { <-ctx.Done() }

func (w *Worker) AddChild() bool {
	_, err := supervise.Start[Worker](w.Children(), supervise.NoArgs{})
	return err == nil
}

type Holder struct {
	Worker *Worker
	Group  *supervise.Group
	Ref    supervise.Ref[Worker]
}

func Share(worker *Worker, group *supervise.Group, ref supervise.Ref[Worker]) Holder {
	other := ref
	return Holder{Worker: worker, Group: group, Ref: other}
}

func Start(group *supervise.Group) (supervise.Ref[Worker], error) {
	parent := group
	return supervise.Start[Worker](parent, supervise.NoArgs{})
}
