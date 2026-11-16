package accepted

import (
	"context"

	"github.com/lucasavila00/immutago/pkg/supervise"
)

type TerminalWorker struct {
	supervise.TerminalService[supervise.NoArgs]
}

func (*TerminalWorker) Run(ctx context.Context) { <-ctx.Done() }

func (w *TerminalWorker) AddChild() bool {
	_, err := supervise.Start[TerminalWorker](w.Children(), supervise.NoArgs{})
	return err == nil
}

func ShareTerminal(worker *TerminalWorker, ref supervise.Ref[TerminalWorker]) *TerminalWorker {
	other := ref
	other.Borrow().AddChild()
	return worker
}
