package api

import (
	"context"

	"github.com/lucasavila00/immutago/pkg/supervise"
)

type Ledger struct {
	supervise.TerminalService[[]int]
}

func (*Ledger) Run(ctx context.Context) { <-ctx.Done() }
func (p *Ledger) Count() int            { return len(p.Args()) }

type Error = error

func (*Ledger) Read() (Error, int, error) { return nil, 1, nil }

type Ref = supervise.Ref[Ledger]

func Start(root *supervise.Group) (Ref, error) { return supervise.Start[Ledger](root, []int{1}) }
