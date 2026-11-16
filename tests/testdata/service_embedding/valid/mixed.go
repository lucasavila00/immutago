package valid

import (
	"context"

	"github.com/lucasavila00/immutago/pkg/supervise"
)

type RetrySelected struct {
	supervise.Service[supervise.NoArgs]
	TerminalDirect
}

func (*RetrySelected) Run(ctx context.Context) { <-ctx.Done() }

type TerminalSelected struct {
	supervise.TerminalService[supervise.NoArgs]
	Direct
}

func (*TerminalSelected) Run(ctx context.Context) { <-ctx.Done() }

var retrySelected supervise.Ref[RetrySelected]
var terminalSelected supervise.Ref[TerminalSelected]

func startMixed(root *supervise.Group) error {
	if _, err := supervise.Start[RetrySelected](root, supervise.NoArgs{}); err != nil {
		return err
	}
	_, err := supervise.Start[TerminalSelected](root, supervise.NoArgs{})
	return err
}
