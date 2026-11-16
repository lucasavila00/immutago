package ledger

import (
	"context"

	"github.com/lucasavila00/immutago/pkg/supervise"
)

type Ledger struct {
	supervise.TerminalService[<-chan int]
}

func (s *Ledger) Run(ctx context.Context) {
	balance := 0
	for {
		select {
		case amount, ok := <-s.Args():
			if !ok {
				return
			}
			balance += amount
			if balance < 0 {
				panic("ledger: balance underflow")
			}
		case <-ctx.Done():
			return
		}
	}
}

func Start(root *supervise.Group, events <-chan int) error {
	_, err := supervise.Start[Ledger](root, events)
	return err
}
