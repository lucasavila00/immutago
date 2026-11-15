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

// ServiceResultReason accepts scalars, frozen strings, service references, arrays, and ByValue structs.
// Fields of a ByValue struct use the ordinary immutable storage proof.
func (p *Prover) ServiceResultReason(t types.Type) string {
	t = types.Unalias(t)
	switch value := t.Underlying().(type) {
	case *types.Basic:
		return p.Reason(t)
	case *types.Array:
		if _, named := t.(*types.Named); !named {
			if reason := p.ServiceResultReason(value.Elem()); reason != "" {
				return "array element: " + reason
			}
			return ""
		}
	case *types.Struct:
		if ByValue(t) || IsLibraryType(t, "ByValue") || IsLibraryType(t, "FrozenString") || IsRuntimeType(t, "Ref") {
			return p.Reason(t)
		}
	case *types.Pointer:
		return "a service result must be a value, not a pointer"
	}
	return "service results require a primitive, im.FrozenString, supervise.Ref, an unnamed array of service results, " +
		"or a struct with im.ByValue"
}
