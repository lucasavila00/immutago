package nested

import (
	"context"

	"github.com/lucasavila00/immutago/pkg/supervise"
)

type TerminalBase struct {
	supervise.TerminalService[supervise.NoArgs]
}
type TerminalBaseAlias = TerminalBase
type TerminalPointer = *TerminalBaseAlias
type TerminalWorker struct{ TerminalPointer }

func (*TerminalWorker) Run(ctx context.Context) { <-ctx.Done() }

var terminalRef supervise.Ref[TerminalWorker]
