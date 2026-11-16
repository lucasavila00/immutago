package pointer

import (
	"context"

	_ "github.com/lucasavila00/immutago/pkg/im"
	"github.com/lucasavila00/immutago/pkg/supervise"
)

type TerminalDirect struct {
	*supervise.TerminalService[supervise.NoArgs]
}

type TerminalMarker = supervise.TerminalService[supervise.NoArgs]
type TerminalPointer = *TerminalMarker
type TerminalWorker struct{ TerminalPointer }

func (*TerminalWorker) Run(ctx context.Context) { <-ctx.Done() }

var terminalRef supervise.Ref[TerminalWorker]

func terminalStart(root *supervise.Group) error {
	_, err := supervise.Start[TerminalWorker](root, supervise.NoArgs{})
	return err
}
