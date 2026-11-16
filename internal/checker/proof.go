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
	repair string
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
	copied   map[types.Type]*obligation
	reads    map[types.Type]*obligation
	queries  Classifier
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
	if p.queries.Locals(t) {
		return p.categoryNode(t, readCategory)
	}
	return p.categoryNode(t, frozenCategory)
}

// categoryNode records one complete type obligation before checking its components.
func (p *proof) categoryNode(t types.Type, category proofCategory) *obligation {
	t = types.Unalias(t)
	table := &p.nodes
	switch category {
	case copiedCategory:
		table = &p.copied
	case readCategory:
		table = &p.reads
	}
	if *table == nil {
		*table = make(map[types.Type]*obligation)
	}
	if existing := (*table)[t]; existing != nil {
		return existing
	}
	result := &obligation{}
	(*table)[t] = result
	if reason := MarkerReason(t); reason != "" {
		result.reason = reason
		result.repair = "embed exactly one of im.Frozen, im.Local, or im.Copied by value"
	} else {
		switch category {
		case frozenCategory:
			p.value(result, t)
		case copiedCategory:
			p.copiedValue(result, t)
		case readCategory:
			p.readValue(result, t)
		}
	}
	if result.reason != "" && result.cause == nil {
		p.failures = append(p.failures, result)
	}
	return result
}

// depend records an owner and handles children that failed in an earlier query.
func (p *proof) depend(owner *obligation, t types.Type, label string) {
	p.dependCategory(owner, t, label, frozenCategory)
}

// dependCategory connects a stored component to its intrinsic category proof.
func (p *proof) dependCategory(owner *obligation, t types.Type, label string, category proofCategory) {
	child := p.categoryNode(t, category)
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
	if IsLibraryType(t, "Cell") {
		result.reason = "Cell slots are mutable"
		result.repair = "keep the Cell in ordinary storage; use its Frozen payload as data"
		return
	}
	if p.queries.Locals(t) {
		result.reason = "Frozen storage cannot contain Local values"
		result.repair = "store Frozen data; keep its Projection or Local result in local bindings"
		return
	}
	if IsLibraryType(t, "Frozen") || IsLibraryType(t, "Copied") {
		result.reason = "immutable markers are annotations, not data"
		result.repair = "embed the marker in a named struct; use data fields for values"
		return
	}
	if IsUncheckedValue(t) {
		return
	}
	if IsRuntimeType(t, "Ref") {
		result.reason = refTargetReason(t.(*types.Named).TypeArgs().At(0))
		result.repair = "use a concrete service struct that embeds supervise.Service by value"
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
		result.repair = "use a concrete proved type for this immutable field or element"
		return
	}
	if _, named := t.(*types.Named); named {
		switch t.Underlying().(type) {
		case *types.Array:
			result.reason = "named array must be wrapped in a struct with an immutable marker"
			result.repair = "use an unnamed array type; to keep a named type, put the unnamed array in a marked struct"
			return
		case *types.Pointer:
			result.reason = "defined pointer types are unsupported in Frozen data; use a pointer alias"
			result.repair = "use a pointer alias such as type Selected = *Settings, where Settings is a Frozen record"
			return
		}
	}
	switch underlying := t.Underlying().(type) {
	case *types.Pointer:
		p.frozenPointer(result, underlying)
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
	case *types.Interface:
		p.frozenInterface(result, t)
		return
	}
	unsupportedFrozen(result, t)
}

// unsupportedFrozen rejects native storage without a Frozen representation.
func unsupportedFrozen(result *obligation, t types.Type) {
	result.reason = fmt.Sprintf("%s is not provably immutable", types.TypeString(t, nil))
	switch t.Underlying().(type) {
	case *types.Slice:
		result.repair = "replace the native slice with im.FrozenSlice[T], where T is a Frozen type"
	case *types.Map:
		result.repair = "replace the native map with im.FrozenMap[K, V], where K and V are Frozen types"
	case *types.Chan, *types.Signature:
		result.repair = "keep channels and callbacks outside immutable data; store their Frozen results"
	default:
		result.repair = "use a type from the Frozen category in docs/reference/types.md"
	}
}

// frozenPointer proves the concrete record target and excludes stored Local pointers.
func (p *proof) frozenPointer(result *obligation, pointer *types.Pointer) {
	if Local(pointer.Elem()) {
		result.reason = "Local pointers cannot be stored"
		result.repair = "keep the Local pointer in a local binding; store a Frozen pointer for shared data"
		return
	}
	if !Marked(pointer.Elem()) {
		result.reason = "stored pointers require a concrete Frozen record target"
		result.repair = "point to a Frozen record, or store the value without a pointer; " +
			"for Copied payloads, use im.Value(projection), im.Value(cell.Load()), or im.Value(slice.At(index))"
		return
	}
	p.depend(result, pointer.Elem(), "")
}

// refTargetReason requires the concrete service value selected by Borrow's *T result.
func refTargetReason(target types.Type) string {
	if _, concrete := target.Underlying().(*types.Struct); !concrete || !Service(target) {
		return "Ref requires a concrete struct that embeds supervise.Service"
	}
	return ServiceEmbeddingReason(target)
}

// structure requires one marker and checks every stored field.
func (p *proof) structure(result *obligation, t types.Type, structure *types.Struct) {
	if Copied(t) {
		p.copiedRecord(result, structure)
		return
	}
	if !Frozen(t) {
		result.reason = "struct does not embed im.Frozen"
		result.repair = "embed im.Frozen and use Frozen fields, or im.Copied for value-only fields"
		return
	}
	for field := range structure.Fields() {
		if field.Embedded() && IsLibraryType(field.Type(), "Frozen") {
			continue
		}
		if field.Embedded() {
			result.reason = field.Name() + ": Frozen data fields must be named"
			result.repair = "give each data field a name; embed only im.Frozen"
			return
		}
		p.depend(result, field.Type(), field.Name()+": ")
	}
}
