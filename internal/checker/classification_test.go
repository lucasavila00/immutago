package checker

import (
	"fmt"
	"go/token"
	"go/types"
	"testing"
)

// classificationStorageOracle retains the original storage walk without shared query state.
func classificationStorageOracle(t types.Type, seen map[types.Type]bool) bool {
	if t == nil || seen[t] {
		return false
	}
	seen[t] = true
	if parameter, ok := types.Unalias(t).(*types.TypeParam); ok {
		return classificationConstraintOracle(parameter.Constraint(), seen)
	}
	if IsLibraryType(t, "Owner") {
		return false
	}
	if IsCollection(t) || Marked(t) {
		return true
	}
	switch u := t.Underlying().(type) {
	case *types.Pointer:
		return classificationStorageOracle(u.Elem(), seen)
	case *types.Array:
		return classificationStorageOracle(u.Elem(), seen)
	case *types.Slice:
		return classificationStorageOracle(u.Elem(), seen)
	case *types.Map:
		return classificationStorageOracle(u.Key(), seen) || classificationStorageOracle(u.Elem(), seen)
	case *types.Chan:
		return classificationStorageOracle(u.Elem(), seen)
	case *types.Struct:
		for field := range u.Fields() {
			if classificationStorageOracle(field.Type(), seen) {
				return true
			}
		}
	}
	return false
}

// classificationConstraintOracle follows the original type-parameter constraint rules.
func classificationConstraintOracle(t types.Type, seen map[types.Type]bool) bool {
	switch u := t.Underlying().(type) {
	case *types.Interface:
		for i := 0; i < u.NumEmbeddeds(); i++ {
			if classificationConstraintOracle(u.EmbeddedType(i), seen) {
				return true
			}
		}
	case *types.Union:
		for i := 0; i < u.Len(); i++ {
			if classificationStorageOracle(u.Term(i).Type(), seen) {
				return true
			}
		}
	default:
		return classificationStorageOracle(t, seen)
	}
	return false
}

// classificationCallableOracle retains the original callback walk and fresh storage queries.
func classificationCallableOracle(t types.Type, seen map[types.Type]bool) bool {
	if t == nil || seen[t] {
		return false
	}
	seen[t] = true
	switch u := t.Underlying().(type) {
	case *types.Signature:
		return classificationSignatureOracle(u, seen)
	case *types.Pointer:
		return classificationCallableOracle(u.Elem(), seen)
	case *types.Array:
		return classificationCallableOracle(u.Elem(), seen)
	case *types.Slice:
		return classificationCallableOracle(u.Elem(), seen)
	case *types.Map:
		return classificationCallableOracle(u.Key(), seen) || classificationCallableOracle(u.Elem(), seen)
	case *types.Chan:
		return classificationCallableOracle(u.Elem(), seen)
	case *types.Struct:
		return classificationCallableFieldsOracle(u, seen)
	case *types.Interface:
		for method := range u.Methods() {
			if classificationCallableOracle(method.Type(), seen) {
				return true
			}
		}
	}
	return false
}

// classificationCallableFieldsOracle checks callbacks in every field with the same visited set.
func classificationCallableFieldsOracle(storage *types.Struct, seen map[types.Type]bool) bool {
	for field := range storage.Fields() {
		if classificationCallableOracle(field.Type(), seen) {
			return true
		}
	}
	return false
}

// classificationSignatureOracle checks signature parameters and results with fresh storage walks.
func classificationSignatureOracle(signature *types.Signature, seen map[types.Type]bool) bool {
	for _, tuple := range []*types.Tuple{signature.Params(), signature.Results()} {
		for item := range tuple.Variables() {
			if classificationStorageOracle(item.Type(), make(map[types.Type]bool)) ||
				classificationCallableOracle(item.Type(), seen) {
				return true
			}
		}
	}
	return false
}

// classificationSignature constructs a concrete function type with one parameter.
func classificationSignature(parameter types.Type) *types.Signature {
	params := types.NewTuple(types.NewParam(token.NoPos, nil, "value", parameter))
	return types.NewSignatureType(nil, nil, nil, params, nil, false)
}

// classificationGraph adds a protected value or callback after the recursive pointer edges.
func classificationGraph(edges [][]int, leaf int, callback bool) []*types.Named {
	marker, _ := testTypes()
	value := markedType("Value", marker)
	nodes := make([]*types.Named, len(edges))
	for i := range nodes {
		name := types.NewTypeName(token.NoPos, nil, fmt.Sprintf("Node%d", i), nil)
		nodes[i] = types.NewNamed(name, types.NewStruct(nil, nil), nil)
	}
	for i, targets := range edges {
		var fields []*types.Var
		for j, target := range targets {
			name := fmt.Sprintf("Edge%d", j)
			fields = append(fields, types.NewField(token.NoPos, nil, name, types.NewPointer(nodes[target]), false))
		}
		if i == leaf {
			var stored types.Type = value
			if callback {
				stored = classificationSignature(value)
			}
			fields = append(fields, types.NewField(token.NoPos, nil, "Protected", stored, false))
		}
		nodes[i].SetUnderlying(types.NewStruct(fields, nil))
	}
	return nodes
}

// TestClassificationCycles checks every root against independent reachability in both query orders.
func TestClassificationCycles(t *testing.T) {
	for mask := 0; mask < 256; mask++ {
		edges := stressPatternEdges(mask)
		for leaf := -1; leaf < len(edges); leaf++ {
			for _, callback := range []bool{false, true} {
				checkClassificationGraph(t, edges, leaf, callback)
			}
		}
	}
}

// checkClassificationGraph checks cached negative results and results inside strongly connected graphs.
func checkClassificationGraph(t *testing.T, edges [][]int, leaf int, callback bool) {
	t.Helper()
	nodes := classificationGraph(edges, leaf, callback)
	for _, reverse := range []bool{false, true} {
		var classifier Classifier
		for index := range nodes {
			root := index
			if reverse {
				root = len(nodes) - 1 - index
			}
			reachable := stressReaches(edges, root, leaf)
			if got, want := classifier.Contains(nodes[root]), reachable && !callback; got != want {
				t.Fatalf("edges=%v leaf=%d root=%d callback=%t reverse=%t: Contains=%t, want %t",
					edges, leaf, root, callback, reverse, got, want)
			}
			if got, want := classifier.Callable(nodes[root]), reachable && callback; got != want {
				t.Fatalf("edges=%v leaf=%d root=%d callback=%t reverse=%t: Callable=%t, want %t",
					edges, leaf, root, callback, reverse, got, want)
			}
		}
	}
}

// classificationShapes supplies mode boundaries, aliases, type sets, and callback storage.
func classificationShapes(t *testing.T) []types.Type {
	t.Helper()
	marker, slice := testTypes()
	value := markedType("Value", marker)
	alias := types.NewAlias(types.NewTypeName(token.NoPos, nil, "Alias", nil), value)
	callback := classificationSignature(alias)
	method := types.NewFunc(token.NoPos, nil, "Read", callback)
	service := types.NewInterfaceType([]*types.Func{method}, nil).Complete()
	terms := []*types.Term{types.NewTerm(true, value.Underlying()), types.NewTerm(false, types.Typ[types.Int])}
	union := types.NewUnion(terms)
	constraint := types.NewInterfaceType(nil, []types.Type{union}).Complete()
	parameter := types.NewTypeParam(types.NewTypeName(token.NoPos, nil, "T", nil), constraint)
	ordinary := types.NewTypeParam(types.NewTypeName(token.NoPos, nil, "U", nil),
		types.NewInterfaceType(nil, nil).Complete())
	localName := types.NewTypeName(token.NoPos, marker.Obj().Pkg(), "Scoped", nil)
	localMarker := types.NewNamed(localName, types.NewStruct(nil, nil), nil)
	local := markedType("Local", localMarker)
	results := types.NewTuple(types.NewParam(token.NoPos, nil, "value", value))
	resultCall := types.NewSignatureType(nil, nil, nil, nil, results, false)
	variadicParams := types.NewTuple(types.NewParam(token.NoPos, nil, "values", types.NewSlice(value)))
	variadic := types.NewSignatureType(nil, nil, nil, variadicParams, nil, true)
	receiver := types.NewParam(token.NoPos, nil, "receiver", types.NewPointer(value))
	receiverOnly := types.NewSignatureType(receiver, nil, nil, nil, nil, false)
	frozen := sliceType(t, slice, value)
	frozenAlias := types.NewAlias(types.NewTypeName(token.NoPos, nil, "FrozenAlias", nil), frozen)
	fakeName := types.NewTypeName(token.NoPos, types.NewPackage("other", "im"), "Shared", nil)
	fakeMarker := types.NewNamed(fakeName, types.NewStruct(nil, nil), nil)
	return []types.Type{
		nil, value, alias, types.NewPointer(alias), types.NewArray(alias, 0), local, types.NewPointer(local),
		callback, types.NewPointer(callback), types.NewArray(callback, 0), types.NewSlice(callback),
		types.NewChan(types.SendRecv, callback), types.NewMap(types.Typ[types.String], callback),
		types.NewMap(types.NewPointer(alias), types.Typ[types.Int]), service,
		constraint, parameter, ordinary, classificationSignature(parameter),
		types.NewTuple(types.NewParam(token.NoPos, nil, "value", value)),
		frozen, frozenAlias, classificationSignature(frozen), resultCall, variadic, receiverOnly,
		classificationSignature(callback), markedType("Pretender", fakeMarker),
		types.NewStruct([]*types.Var{types.NewField(token.NoPos, nil, "Call", callback, false)}, nil),
	}
}

// TestClassificationMatchesWalkers compares all source shapes with the original independent walkers.
func TestClassificationMatchesWalkers(t *testing.T) {
	shapes := classificationShapes(t)
	for _, reverse := range []bool{false, true} {
		var classifier Classifier
		for index := range shapes {
			query := index
			if reverse {
				query = len(shapes) - 1 - index
			}
			typ := shapes[query]
			wantStored := classificationStorageOracle(typ, make(map[types.Type]bool))
			wantCallable := classificationCallableOracle(typ, make(map[types.Type]bool))
			// Query callback mode first to check cross-mode sharing as well as ordinary storage-first queries.
			if got := classifier.Callable(typ); got != wantCallable {
				t.Fatalf("shape=%d reverse=%t: Callable=%t, want %t", query, reverse, got, wantCallable)
			}
			if got := classifier.Contains(typ); got != wantStored {
				t.Fatalf("shape=%d reverse=%t: Contains=%t, want %t", query, reverse, got, wantStored)
			}
		}
	}
}

// TestClassifierOwnerIsOpaque checks that slot implementation fields are not immutable storage.
func TestClassifierOwnerIsOpaque(t *testing.T) {
	marker, _ := testTypes()
	value := markedType("Value", marker)
	field := types.NewField(token.NoPos, marker.Obj().Pkg(), "value", types.NewPointer(value), false)
	name := types.NewTypeName(token.NoPos, marker.Obj().Pkg(), "Owner", nil)
	owner := types.NewNamed(name, types.NewStruct([]*types.Var{field}, nil), nil)
	var queries Classifier
	for _, typ := range []types.Type{owner, types.NewPointer(owner), types.NewArray(owner, 1)} {
		if queries.Contains(typ) || Contains(typ) {
			t.Errorf("Owner implementation is not opaque for %s", typ)
		}
	}
}
