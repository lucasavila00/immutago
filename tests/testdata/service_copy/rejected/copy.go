package rejected

import (
	"context"
	"sync"

	"github.com/lucasavila00/immutago/pkg/supervise"
)

type Owner struct{ Supervisor supervise.Supervisor }

func TakeGroup(supervise.Supervisor) {}
func TakeOwner(Owner)                {}

func CopyGroup(group *supervise.Supervisor) {
	copy := *group
	TakeGroup(copy)
	owner := Owner{Supervisor: *group}
	TakeOwner(owner)
}

func ReturnGroup(group *supervise.Supervisor) supervise.Supervisor { return *group }

type LockedWorker struct {
	supervise.Service[supervise.NoArgs]
	mu sync.Mutex
}

func (*LockedWorker) Run(ctx context.Context) { <-ctx.Done() }

func CopyLockedWorker(worker *LockedWorker) { copy := *worker; _ = &copy }

type Args struct{ mu sync.Mutex }
type ArgsWorker struct{ supervise.Service[Args] }

func (*ArgsWorker) Run(ctx context.Context) { <-ctx.Done() }

func CopyArgsWorker(worker *ArgsWorker) { copy := *worker; _ = &copy }
