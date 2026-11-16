package checker

import (
	"go/types"
	"strings"
)

// Prover reuses immutable type obligations within one analysis pass.
// Its zero value is ready for use. Types must be complete and must not change.
// A Prover must not be used by concurrent passes.
type Prover struct {
	graph    proof
	failures map[*obligation]TypeFailure
}

// Reason proves t or returns a component path to a failure.
// The selected cause can depend on earlier queries; each root message is cached.
func (p *Prover) Reason(t types.Type) string { return p.render(p.graph.node(t)).Reason }

// FrozenReason proves the shared payload contract used by Cell and Projection.
func (p *Prover) FrozenReason(t types.Type) string {
	return p.FrozenFailure(t).Reason
}

// CopiedReason proves bool, numbers, Copied records, unnamed arrays, or opaque service references.
func (p *Prover) CopiedReason(t types.Type) string {
	return p.CopiedFailure(t).Reason
}

// ReadResultReason proves Copied values, local values, and unchecked handles through one type rule.
func (p *Prover) ReadResultReason(t types.Type) string {
	return p.ReadResultFailure(t).Reason
}

// TypeFailure contains one component path and the repair supplied by its failing rule.
// An empty Reason means that the proof passed.
type TypeFailure struct {
	Reason string
	Repair string
}

// render completes pending obligations and reuses the selected failure path.
func (p *Prover) render(root *obligation) TypeFailure {
	p.graph.propagate()
	if !root.failed() {
		return TypeFailure{}
	}
	if failure, found := p.failures[root]; found {
		return failure
	}
	var message strings.Builder
	var repair string
	for current := root; current != nil; current = current.cause {
		message.WriteString(current.reason)
		repair = current.repair
	}
	failure := TypeFailure{Reason: message.String(), Repair: repair}
	if p.failures == nil {
		p.failures = make(map[*obligation]TypeFailure)
	}
	p.failures[root] = failure
	return failure
}

// FrozenFailure returns the failed type rule and its repair from the same proof.
func (p *Prover) FrozenFailure(t types.Type) TypeFailure {
	return p.render(p.graph.categoryNode(t, frozenCategory))
}

// CopiedFailure returns the failed type rule and its repair from the same proof.
func (p *Prover) CopiedFailure(t types.Type) TypeFailure {
	return p.render(p.graph.categoryNode(t, copiedCategory))
}

// ReadResultFailure returns the failed type rule and its repair from the same proof.
func (p *Prover) ReadResultFailure(t types.Type) TypeFailure {
	return p.render(p.graph.categoryNode(t, readCategory))
}
