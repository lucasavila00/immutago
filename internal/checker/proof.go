package checker

import (
	"fmt"
	"go/types"
)

// proofKey separates a stored interface value from a generic constraint with the same type.
type proofKey struct {
	typ        types.Type
	constraint bool
}

// obligation records an AND or OR node in the structural proof graph.
type obligation struct {
	users        []proofUse
	alternatives bool
	dependencies int
	failed       int
	reason       string
}

// proofUse identifies an owner and its diagnostic path from a component obligation.
type proofUse struct {
	owner *obligation
	label string
}

// proof stores one query's graph so shared type branches are visited only once.
// Storage queries prove absence of immutable storage; other queries prove immutability.
type proof struct {
	nodes    map[proofKey]*obligation
	failures []*obligation
	storage  bool
}

// Reason returns an empty string when t is proved immutable, or the first failed obligation.
// Recursive components start provisionally valid. Definite failures propagate
// through all owners, including cycles, in linear time in the type graph.
func Reason(t types.Type) string {
	p := &proof{nodes: make(map[proofKey]*obligation)}
	return p.evaluate(t)
}

// PossibleStorage reports whether a type can contain immutable storage, including unresolved generic fields.
// Ordinary interface values contain copies and do not imply stored immutable fields.
func PossibleStorage(t types.Type) bool {
	if t == nil {
		return false
	}
	p := &proof{nodes: make(map[proofKey]*obligation), storage: true}
	return p.evaluate(t) != ""
}

// evaluate propagates each failed component to its owners after the graph is complete.
func (p *proof) evaluate(t types.Type) string {
	root := p.node(t, false)
	for index := 0; index < len(p.failures); index++ {
		child := p.failures[index]
		for _, use := range child.users {
			owner := use.owner
			if owner.reason != "" {
				continue
			}
			owner.failed++
			if owner.alternatives && owner.failed != owner.dependencies {
				continue
			}
			owner.reason = use.label + child.reason
			if owner.alternatives {
				owner.reason = "type parameter constraint does not prove immutability"
			}
			p.failures = append(p.failures, owner)
		}
	}
	return root.reason
}

// node creates one obligation before descending, which terminates recursive type graphs.
func (p *proof) node(t types.Type, constraint bool) *obligation {
	t = types.Unalias(t)
	key := proofKey{typ: t, constraint: constraint}
	if existing := p.nodes[key]; existing != nil {
		return existing
	}
	result := &obligation{}
	p.nodes[key] = result
	if constraint {
		p.constraint(result, t)
	} else if p.storage {
		p.storageValue(result, t)
	} else {
		p.value(result, t)
	}
	if result.reason != "" {
		p.failures = append(p.failures, result)
	}
	return result
}

// depend adds a component edge and its reverse propagation edge.
func (p *proof) depend(owner *obligation, t types.Type, constraint bool, label string) {
	owner.dependencies++
	child := p.node(t, constraint)
	child.users = append(child.users, proofUse{owner: owner, label: label})
}

// value builds the obligations for one stored value using its Go compiler type.
func (p *proof) value(result *obligation, t types.Type) {
	if IsLibraryType(t, "Marker") {
		return
	}
	if IsCollection(t) {
		args := t.(*types.Named).TypeArgs()
		for i := 0; i < args.Len(); i++ {
			p.depend(result, args.At(i), false, fmt.Sprintf("type argument %d: ", i+1))
		}
		return
	}
	if parameter, ok := t.(*types.TypeParam); ok {
		p.depend(result, parameter.Constraint(), true, "")
		return
	}
	switch underlying := t.Underlying().(type) {
	case *types.Basic:
		if underlying.Info()&(types.IsBoolean|types.IsInteger|types.IsFloat|types.IsComplex|types.IsString) != 0 {
			return
		}
	case *types.Array:
		p.depend(result, underlying.Elem(), false, "")
		return
	case *types.Struct:
		p.structure(result, t, underlying)
		return
	}
	result.reason = fmt.Sprintf("%s is not provably immutable", types.TypeString(t, nil))
}

// structure requires the marker and creates an obligation for every field.
func (p *proof) structure(result *obligation, t types.Type, structure *types.Struct) {
	if !Marked(t) {
		result.reason = "struct does not embed immutable.Marker"
		return
	}
	for field := range structure.Fields() {
		p.depend(result, field.Type(), false, field.Name()+": ")
	}
}

// constraint models intersected bounds as alternatives: any safe bound proves the intersection safe.
// Every union member must satisfy the query. Empty or method-only bounds prove nothing.
func (p *proof) constraint(result *obligation, t types.Type) {
	switch bound := t.Underlying().(type) {
	case *types.Interface:
		result.alternatives = true
		for i := 0; i < bound.NumEmbeddeds(); i++ {
			p.depend(result, bound.EmbeddedType(i), true, "")
		}
		if result.dependencies == 0 {
			result.reason = "type parameter constraint does not prove immutability"
		}
	case *types.Union:
		for i := 0; i < bound.Len(); i++ {
			p.depend(result, bound.Term(i).Type(), false, "")
		}
	default:
		p.depend(result, t, false, "")
	}
}

// storageValue proves that a storage type excludes marked values for every permitted instantiation.
func (p *proof) storageValue(result *obligation, t types.Type) {
	if Marked(t) || IsCollection(t) {
		result.reason = "contains immutable storage"
		return
	}
	if parameter, ok := t.(*types.TypeParam); ok {
		p.depend(result, parameter.Constraint(), true, "")
		return
	}
	switch underlying := t.Underlying().(type) {
	case *types.Pointer:
		p.depend(result, underlying.Elem(), false, "")
	case *types.Array:
		p.depend(result, underlying.Elem(), false, "")
	case *types.Slice:
		p.depend(result, underlying.Elem(), false, "")
	case *types.Chan:
		p.depend(result, underlying.Elem(), false, "")
	case *types.Map:
		p.depend(result, underlying.Key(), false, "")
		p.depend(result, underlying.Elem(), false, "")
	case *types.Struct:
		for field := range underlying.Fields() {
			p.depend(result, field.Type(), false, field.Name()+": ")
		}
	}
}
