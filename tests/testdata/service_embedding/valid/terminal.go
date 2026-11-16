package valid

import (
	"context"

	"github.com/lucasavila00/immutago/pkg/supervise"
)

type TerminalMarker = supervise.TerminalService[supervise.NoArgs]
type TerminalDirect struct{ TerminalMarker }

func (*TerminalDirect) Run(ctx context.Context) { <-ctx.Done() }
func (*TerminalDirect) Count() int              { return 1 }

type TerminalBase = TerminalDirect
type TerminalMiddle struct{ TerminalBase }
type TerminalState struct{ Count int }
type TerminalNested struct {
	TerminalMiddle
	*TerminalState
}

func (*TerminalNested) Run(ctx context.Context) { <-ctx.Done() }

type TerminalNestedAlias = TerminalNested

type TerminalDefined TerminalDirect

func (*TerminalDefined) Run(ctx context.Context) { <-ctx.Done() }

type TerminalRecursive struct {
	TerminalMarker
	*TerminalRecursive
}

func (*TerminalRecursive) Run(ctx context.Context) { <-ctx.Done() }

type TerminalAnonymous = struct{ TerminalDirect }

var terminalDefined supervise.Ref[TerminalDefined]
var terminalRecursive supervise.Ref[TerminalRecursive]
var terminalAnonymous supervise.Ref[TerminalAnonymous]

var terminalDirect supervise.Ref[TerminalDirect]
var terminalNested supervise.Ref[TerminalNestedAlias]

func terminalStart(root *supervise.Group) error {
	var terminalErr error
	terminalDirect, terminalErr = supervise.Start[TerminalDirect](root, supervise.NoArgs{})
	if terminalErr != nil {
		return terminalErr
	}
	_, terminalErr = supervise.Start[TerminalNestedAlias](root, supervise.NoArgs{})
	return terminalErr
}

func terminalRead() int { return terminalDirect.Borrow().Count() }
