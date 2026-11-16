package valid

import (
	"context"

	"example.com/consumer/api"
	"github.com/lucasavila00/immutago/pkg/im"
	"github.com/lucasavila00/immutago/pkg/supervise"
)

type Reply struct {
	im.Local
	Text im.StringProjection
}

type Worker struct {
	supervise.TerminalService[im.FrozenString]
}

func (*Worker) Run(ctx context.Context) { <-ctx.Done() }
func (p *Worker) Read() (Reply, bool, error) {
	return Reply{Text: p.Args().Projection()}, true, nil
}
func (*Worker) Reference() api.Ref { return api.Ref{} }

type LocalRef struct {
	im.Local
	Value api.Ref
}

var saved api.Ref
var collection im.FrozenSlice[api.Ref]
var found bool

func identity[T any](value T) T { return value }

func Read(ref supervise.Ref[Worker], ordinary api.Ref) (Reply, error) {
	var reply Reply
	reply, found, _ = ref.Borrow().Read()
	var err error
	err, count, _ := ordinary.Borrow().Read()
	_ = count
	saved = identity(ordinary)
	var boxed any = ordinary
	_ = boxed
	borrower := interface{ Borrow() *api.Ledger }(ordinary)
	borrower.Borrow().Count()
	return reply, err
}

func Start(root *supervise.Group) error {
	_, err := supervise.Start[Worker](root, im.FreezeString("ready"))
	return err
}
