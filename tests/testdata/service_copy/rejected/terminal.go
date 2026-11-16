package rejected

import (
	"context"

	"github.com/lucasavila00/immutago/pkg/supervise"
)

type TerminalWorker struct {
	supervise.TerminalService[supervise.NoArgs]
}

func (*TerminalWorker) Run(ctx context.Context) { <-ctx.Done() }
func (w TerminalWorker) Ready() bool            { return true }

type TerminalAlias = TerminalWorker
type TerminalNested struct{ TerminalAlias }

func (w TerminalNested) Count() int { return 1 }

func TakeTerminal(TerminalWorker) {}

func CopyTerminal(worker *TerminalWorker) {
	copy := *worker
	TakeTerminal(copy)
	var other TerminalWorker = *worker
	TakeTerminal(other)
	values := []TerminalWorker{*worker}
	for _, value := range values {
		TakeTerminal(value)
	}
}

func ReturnTerminal(worker *TerminalWorker) TerminalWorker { return *worker }
