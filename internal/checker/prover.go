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
	failures map[*obligation]Failure
}

// Reason proves t or returns a component path to a failure.
// The selected cause can depend on earlier queries; each root message is cached.
func (p *Prover) Reason(t types.Type) string { return p.Failure(t).Reason }

// Failure returns the failed storage rule and its repair from the same proof.
func (p *Prover) Failure(t types.Type) Failure { return p.render(p.graph.node(t)) }

// FrozenReason proves the shared payload contract used by Cell and Projection.
func (p *Prover) FrozenReason(t types.Type) string {
	return p.FrozenFailure(t).Reason
}

// CopiedReason proves bool, numbers, Copied records, or unnamed arrays.
func (p *Prover) CopiedReason(t types.Type) string {
	return p.CopiedFailure(t).Reason
}

// ReadResultReason proves Copied values, local values, and opaque handles through one type rule.
func (p *Prover) ReadResultReason(t types.Type) string {
	return p.ReadResultFailure(t).Reason
}

// Failure contains a rejected rule and the repair supplied by that rule.
// An empty Reason means that the proof passed.
type Failure struct {
	Reason string
	Repair string
}

// render completes pending obligations and reuses the selected failure path.
func (p *Prover) render(root *obligation) Failure {
	p.graph.propagate()
	if !root.failed() {
		return Failure{}
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
	failure := Failure{Reason: message.String(), Repair: repair}
	if p.failures == nil {
		p.failures = make(map[*obligation]Failure)
	}
	p.failures[root] = failure
	return failure
}

// FrozenFailure returns the failed type rule and its repair from the same proof.
func (p *Prover) FrozenFailure(t types.Type) Failure {
	return p.render(p.graph.categoryNode(t, frozenCategory))
}

// CopiedFailure returns the failed type rule and its repair from the same proof.
func (p *Prover) CopiedFailure(t types.Type) Failure {
	return p.render(p.graph.categoryNode(t, copiedCategory))
}

// ReadResultFailure returns the failed type rule and its repair from the same proof.
func (p *Prover) ReadResultFailure(t types.Type) Failure {
	return p.render(p.graph.categoryNode(t, readCategory))
}
