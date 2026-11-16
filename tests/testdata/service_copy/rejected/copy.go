package rejected

import (
	"context"

	"github.com/lucasavila00/immutago/pkg/supervise"
)

type Worker struct {
	supervise.Service[supervise.NoArgs]
}

func (*Worker) Run(ctx context.Context) { <-ctx.Done() }

func (w Worker) AddChild() bool {
	_, err := supervise.Start[Worker](w.UnsafeChildren(), supervise.NoArgs{})
	return err == nil
}

type Alias = Worker
type Nested struct{ Alias }
type Owner struct{ Group supervise.Group }

func (w Nested) Ready() bool { return true }

func TakeWorker(Worker)         {}
func TakeGroup(supervise.Group) {}
func TakeOwner(Owner)           {}

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

func CopyGroup(group *supervise.Group) {
	copy := *group
	TakeGroup(copy)
	owner := Owner{Group: *group}
	TakeOwner(owner)
}

func ReturnWorker(worker *Worker) Worker                 { return *worker }
func ReturnGroup(group *supervise.Group) supervise.Group { return *group }
