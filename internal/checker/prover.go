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
func (p *Prover) Reason(t types.Type) string {
	if p.graph.nodes == nil {
		p.graph.nodes = make(map[types.Type]*obligation)
	}
	root := p.graph.node(t)
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

// ArgumentReason proves a process argument type. It accepts proc.Dependency as the type, or as a TrustedField of a
// struct with exactly one marker. Every other component must be an allowed stored type.
func (p *Prover) ArgumentReason(t types.Type) string {
	if IsRuntimeType(t, DependencyType) {
		return ""
	}
	structure, ok := t.Underlying().(*types.Struct)
	if !ok || MarkerCount(t) != 1 {
		return p.Reason(t)
	}
	for field := range structure.Fields() {
		if TrustedField(field) {
			continue
		}
		if reason := p.Reason(field.Type()); reason != "" {
			return field.Name() + ": " + reason
		}
	}
	return ""
}
