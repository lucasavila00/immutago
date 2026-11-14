package process

import (
	"context"

	"github.com/lucasavila00/immutago/pkg/proc"
)

// OrdinaryBody demonstrates the process boundary, without memory isolation.
type OrdinaryBody struct {
	proc.Process[proc.None]
	count int
	table map[string]int
}

// Run can use ordinary mutable field handles.
func (p *OrdinaryBody) Run(ctx context.Context) {
	counter := &p.count
	*counter++
	p.table = map[string]int{}
	<-ctx.Done()
}
