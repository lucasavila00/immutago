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

// Reason proves t or returns its first failed component, with the same ordering as a fresh proof.
// Validity is shared across queries; exact diagnostics are built once for each invalid root.
func (p *Prover) Reason(t types.Type) string {
	if p.graph.nodes == nil {
		p.graph.nodes = make(map[types.Type]*obligation)
	}
	root := p.graph.node(t)
	p.graph.propagate()
	if !root.failed {
		return ""
	}
	if reason, found := p.reasons[root]; found {
		return reason
	}
	replay := diagnosticReplay{nodes: make(map[*obligation]*proofDiagnostic)}
	reason := replay.reason(root)
	if p.reasons == nil {
		p.reasons = make(map[*obligation]string)
	}
	p.reasons[root] = reason
	return reason
}

// proofDiagnostic retains a query-local terminal error or a label and first failure cause.
type proofDiagnostic struct {
	users []diagnosticUse
	text  string
	cause *proofDiagnostic
}

// diagnosticUse identifies one query-local owner and the edge label leading to it.
type diagnosticUse struct {
	owner *proofDiagnostic
	label string
}

// diagnosticReplay restores the fresh depth-first discovery and FIFO failure order for one root.
type diagnosticReplay struct {
	nodes    map[*obligation]*proofDiagnostic
	failures []*proofDiagnostic
}

// node visits ordered failed components before it adds their reverse edges, as the fresh proof did.
func (r *diagnosticReplay) node(source *obligation) *proofDiagnostic {
	if existing := r.nodes[source]; existing != nil {
		return existing
	}
	result := &proofDiagnostic{text: source.reason}
	r.nodes[source] = result
	for _, part := range source.parts {
		if !part.node.failed {
			continue
		}
		child := r.node(part.node)
		child.users = append(child.users, diagnosticUse{owner: result, label: part.label})
	}
	if result.text != "" {
		r.failures = append(r.failures, result)
	}
	return result
}

// reason propagates the first query-local cause and renders only the requested path.
func (r *diagnosticReplay) reason(source *obligation) string {
	root := r.node(source)
	for index := 0; index < len(r.failures); index++ {
		child := r.failures[index]
		for _, use := range child.users {
			if use.owner.text == "" && use.owner.cause == nil {
				use.owner.text, use.owner.cause = use.label, child
				r.failures = append(r.failures, use.owner)
			}
		}
	}
	var message strings.Builder
	for root.cause != nil {
		message.WriteString(root.text)
		root = root.cause
	}
	message.WriteString(root.text)
	return message.String()
}
