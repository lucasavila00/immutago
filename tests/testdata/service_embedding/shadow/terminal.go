package shadow

import (
	"context"

	"github.com/lucasavila00/immutago/pkg/supervise"
)

type TerminalBase struct {
	supervise.TerminalService[supervise.NoArgs]
}
type TerminalMiddle struct{ TerminalBase }
type TerminalWorker struct {
	TerminalMiddle
	*supervise.TerminalService[supervise.NoArgs]
}

func (*TerminalWorker) Run(ctx context.Context) { <-ctx.Done() }

var terminalRef supervise.Ref[TerminalWorker]
