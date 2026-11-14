package checker

import (
	"fmt"
	"go/types"
	"strings"
)

// originalObligation records a terminal reason or an edge label with its first failure cause.
// Each cause points to an already failed node; only unfailed owners get causes.
// Thus cause links cannot form a cycle.
type originalObligation struct {
	users  []originalProofUse
	reason string
	cause  *originalObligation
}

// originalProofUse identifies an owner and its diagnostic path from a component originalObligation.
type originalProofUse struct {
	owner *originalObligation
	label string
}

// originalProof visits each concrete type once and propagates failures through recursive owners.
type originalProof struct {
	nodes    map[types.Type]*originalObligation
	failures []*originalObligation
}

// originalReason is the fresh proof implementation before pass-local graph reuse.
func originalReason(t types.Type) string {
	p := &originalProof{nodes: make(map[types.Type]*originalObligation)}
	root := p.node(t)
	for index := 0; index < len(p.failures); index++ {
		child := p.failures[index]
		for _, use := range child.users {
			if use.owner.reason == "" && use.owner.cause == nil {
				use.owner.reason = use.label
				use.owner.cause = child
				p.failures = append(p.failures, use.owner)
			}
		}
	}
	if root.cause == nil {
		return root.reason
	}
	var message strings.Builder
	for root.cause != nil {
		message.WriteString(root.reason)
		root = root.cause
	}
	message.WriteString(root.reason)
	return message.String()
}

// node records an originalObligation before descending, so recursive references terminate.
func (p *originalProof) node(t types.Type) *originalObligation {
	t = types.Unalias(t)
	if existing := p.nodes[t]; existing != nil {
		return existing
	}
	result := &originalObligation{}
	p.nodes[t] = result
	p.value(result, t)
	if result.reason != "" {
		p.failures = append(p.failures, result)
	}
	return result
}

// depend adds a component and its reverse failure-propagation edge.
func (p *originalProof) depend(owner *originalObligation, t types.Type, label string) {
	child := p.node(t)
	child.users = append(child.users, originalProofUse{owner: owner, label: label})
}

// value matches the complete concrete immutable storage forms.
func (p *originalProof) value(result *originalObligation, t types.Type) {
	if IsLibraryType(t, "Owner") {
		result.reason = "Owner slots are mutable"
		return
	}
	if IsLibraryType(t, "Shared") || IsLibraryType(t, "Scoped") {
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
func (p *originalProof) structure(result *originalObligation, t types.Type, structure *types.Struct) {
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
