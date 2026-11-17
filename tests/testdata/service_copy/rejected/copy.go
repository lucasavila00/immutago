package rejected

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

func (w Worker) AddChild() bool {
	_, err := supervise.Start[Worker](w.supervisor, supervise.NoArgs{})
	return err == nil
}

type Alias = Worker
type Nested struct{ Alias }
type Owner struct{ Supervisor supervise.Supervisor }

func (w Nested) Ready() bool { return true }

func TakeWorker(Worker)              {}
func TakeGroup(supervise.Supervisor) {}
func TakeOwner(Owner)                {}

func CopyService(worker *Worker) {
	copy := *worker
	TakeWorker(copy)
	var other Worker = *worker
	TakeWorker(other)
	values := []Worker{*worker}
	for _, value := range values {
		TakeWorker(value)
	}
}

func CopyGroup(group *supervise.Supervisor) {
	copy := *group
	TakeGroup(copy)
	owner := Owner{Supervisor: *group}
	TakeOwner(owner)
}

func ReturnWorker(worker *Worker) Worker                           { return *worker }
func ReturnGroup(group *supervise.Supervisor) supervise.Supervisor { return *group }
