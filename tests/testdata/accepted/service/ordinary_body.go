package service

import (
	"context"

	"github.com/lucasavila00/immutago/pkg/supervise"
)

var sharedCount *int
var sharedTable = map[string]int{}

// OrdinaryBody demonstrates the four-rule boundary, without memory isolation.
type OrdinaryBody struct {
	supervise.Service[supervise.NoArgs]
	count int
	table map[string]int
}

// Run can use ordinary mutable globals and field handles.
func (p *OrdinaryBody) Run(ctx context.Context) {
	sharedCount = &p.count
	p.table = sharedTable
	<-ctx.Done()
}
