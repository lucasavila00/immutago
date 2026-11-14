package checker

import (
	"fmt"
	"go/types"
)

// obligation retains a terminal reason or a label with an earlier failure cause.
// Only an unfailed owner can get a cause, so cause links cannot form a cycle.
type obligation struct {
	users  []proofUse
	reason string
	cause  *obligation
}

// proofUse identifies an owner and the component label on its reverse edge.
type proofUse struct {
	owner *obligation
	label string
}

// proofSeed retains a new edge to a child whose failure was already known.
type proofSeed struct {
	owner *obligation
	child *obligation
	label string
}

// failed includes empty-label pointer failures as well as terminal errors.
func (n *obligation) failed() bool { return n.reason != "" || n.cause != nil }

// proof visits each concrete type once and propagates failures through recursive owners.
type proof struct {
	nodes    map[types.Type]*obligation
	failures []*obligation
	pending  []proofSeed
}

// Reason proves concrete immutable storage or returns a failed component.
func Reason(t types.Type) string {
	var prover Prover
	return prover.Reason(t)
}

// node records an obligation before descending, so recursive references terminate.
func (p *proof) node(t types.Type) *obligation {
	t = types.Unalias(t)
	if existing := p.nodes[t]; existing != nil {
		return existing
	}
	result := &obligation{}
	p.nodes[t] = result
	p.value(result, t)
	if result.reason != "" && result.cause == nil {
		p.failures = append(p.failures, result)
	}
	return result
}

// depend records an owner and handles children that failed in an earlier query.
func (p *proof) depend(owner *obligation, t types.Type, label string) {
	child := p.node(t)
	child.users = append(child.users, proofUse{owner: owner, label: label})
	if child.failed() {
		p.pending = append(p.pending, proofSeed{owner: owner, child: child, label: label})
	}
}

// fail links an unfailed owner to an earlier failure with nonempty text.
// Skipping an empty-label child bounds rendering work by the message length.
func (p *proof) fail(owner, child *obligation, label string) {
	if !owner.failed() {
		if child.reason == "" {
			child = child.cause
		}
		owner.reason, owner.cause = label, child
		p.failures = append(p.failures, owner)
	}
}

// propagate prefers terminal failures before it seeds paths through earlier failed children.
func (p *proof) propagate() {
	for _, seed := range p.pending {
		if seed.child.cause == nil {
			p.fail(seed.owner, seed.child, seed.label)
		}
	}
	p.drainFailures()
	for _, seed := range p.pending {
		p.fail(seed.owner, seed.child, seed.label)
	}
	p.drainFailures()
	p.pending = p.pending[:0]
}

// drainFailures visits each newly failed owner once after graph expansion is complete.
func (p *proof) drainFailures() {
	for index := 0; index < len(p.failures); index++ {
		child := p.failures[index]
		for _, use := range child.users {
			p.fail(use.owner, child, use.label)
		}
	}
	p.failures = p.failures[:0]
}

// value matches the complete concrete immutable storage forms.
func (p *proof) value(result *obligation, t types.Type) {
	if IsLibraryType(t, "Owner") {
		result.reason = "Owner slots are mutable"
		return
	}
	if IsRuntimeType(t, UncheckedDependencyType) {
		result.reason = "proc.UncheckedDependency is allowed only as a process argument type or a field of one"
		return
	}
	if IsLibraryType(t, "Shared") || IsLibraryType(t, "Scoped") || IsRuntimeType(t, PIDType) {
		return
	}
	if IsCollection(t) {
		args := t.(*types.Named).TypeArgs()
		for index := 0; index < args.Len(); index++ {
			p.depend(result, args.At(index), fmt.Sprintf("type argument %d: ", index+1))
		}
		return
	}
	if _, generic := t.(*types.TypeParam); generic {
		result.reason = "type parameters are unsupported in immutable storage"
		return
	}
	if _, named := t.(*types.Named); named {
		if _, array := t.Underlying().(*types.Array); array {
			result.reason = "named array must be wrapped in a struct with an immutable marker"
			return
		}
	}
	switch underlying := t.Underlying().(type) {
	case *types.Pointer:
		if Scoped(underlying.Elem()) {
			result.reason = "Scoped pointers cannot be stored"
			return
		}
		if !Marked(underlying.Elem()) {
			result.reason = "stored pointers require a concrete marked struct target"
			return
		}
		p.depend(result, underlying.Elem(), "")
		return
	case *types.Basic:
		if underlying.Info()&(types.IsBoolean|types.IsInteger|types.IsFloat|types.IsComplex|types.IsString) != 0 {
			return
		}
	case *types.Array:
		p.depend(result, underlying.Elem(), "")
		return
	case *types.Struct:
		p.structure(result, t, underlying)
		return
	}
	result.reason = fmt.Sprintf("%s is not provably immutable", types.TypeString(t, nil))
}

// structure requires one marker and checks every stored field.
func (p *proof) structure(result *obligation, t types.Type, structure *types.Struct) {
	if !Marked(t) {
		result.reason = "struct does not embed im.Shared or im.Scoped"
		return
	}
	if Shared(t) && Scoped(t) {
		result.reason = "struct cannot embed both im.Shared and im.Scoped"
		return
	}
	if MarkerCount(t) != 1 {
		result.reason = "struct must embed exactly one immutable marker"
		return
	}
	for field := range structure.Fields() {
		p.depend(result, field.Type(), field.Name()+": ")
	}
}
