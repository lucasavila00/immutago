package checker

import (
	"go/types"
	"strings"
)

// Prover reuses immutable type obligations within one analysis pass.
// Its zero value is ready for use. Types must be complete and must not change.
// A Prover must not be used by concurrent passes.
type Prover struct {
	graph   proof
	reasons map[*obligation]string
}

// Reason proves t or returns a component path to a failure.
// The selected cause can depend on earlier queries; each root message is cached.
func (p *Prover) Reason(t types.Type) string { return p.render(p.graph.node(t)) }

// FrozenReason proves the shared payload contract used by Cell and Projection.
func (p *Prover) FrozenReason(t types.Type) string {
	return p.render(p.graph.categoryNode(t, frozenCategory))
}

// CopiedReason proves that a value can leave a projection without a shared data reference.
// Opaque service references retain their explicit reference contract.
func (p *Prover) CopiedReason(t types.Type) string {
	return p.render(p.graph.categoryNode(t, copiedCategory))
}

// ReadResultReason proves Copied values and local read values through one type rule.
func (p *Prover) ReadResultReason(t types.Type) string {
	return p.render(p.graph.categoryNode(t, readCategory))
}

// render completes pending obligations and reuses the selected failure path.
func (p *Prover) render(root *obligation) string {
	p.graph.propagate()
	if !root.failed() {
		return ""
	}
	if reason, found := p.reasons[root]; found {
		return reason
	}
	var message strings.Builder
	for current := root; current != nil; current = current.cause {
		message.WriteString(current.reason)
	}
	reason := message.String()
	if p.reasons == nil {
		p.reasons = make(map[*obligation]string)
	}
	p.reasons[root] = reason
	return reason
}
