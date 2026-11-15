package pointer

import (
	"context"

	_ "github.com/lucasavila00/immutago/pkg/im"
	"github.com/lucasavila00/immutago/pkg/supervise"
)

type Direct struct {
	*supervise.Service[supervise.NoArgs]
}

type Marker = supervise.Service[supervise.NoArgs]
type Pointer = *Marker
type Worker struct{ Pointer }

func (*Worker) Run(ctx context.Context) { <-ctx.Done() }

var ref supervise.Ref[Worker]

func start(root *supervise.Group) error {
	_, err := supervise.Start[Worker](root, supervise.NoArgs{})
	return err
}
