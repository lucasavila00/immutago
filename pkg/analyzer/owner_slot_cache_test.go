package analyzer

import (
	"go/token"
	"go/types"
	"testing"
)

// TestSlotCacheReusesCompletedTypes prevents fresh wrappers from expanding a known negative cycle again.
func TestSlotCacheReusesCompletedTypes(t *testing.T) {
	pass := sourcePass(t, `type Checked struct{im.Shared}
 type A struct{ Next []B; N int }
 type B struct{ Next []A }`)
	a := pass.Pkg.Scope().Lookup("A").Type()
	b := pass.Pkg.Scope().Lookup("B").Type()
	c := &subsetChecks{}
	if c.hasSlotValue(a) || c.hasSlotValue(types.Typ[types.String]) {
		t.Fatal("ordinary storage contains an Owner slot")
	}
	variable := func(typ types.Type) *types.Var { return types.NewVar(token.NoPos, nil, "", typ) }
	wrappers := []types.Type{
		types.NewArray(a, 1), types.NewSlice(a), types.NewMap(types.Typ[types.String], a),
		types.NewChan(types.SendRecv, a), types.NewPointer(a), types.NewTuple(variable(a)),
		types.NewStruct([]*types.Var{variable(a)}, nil),
		types.NewSignatureType(nil, nil, nil, types.NewTuple(variable(a)), types.NewTuple(variable(b)), false),
	}
	for _, wrapper := range wrappers {
		seen := make(map[types.Type]bool)
		if c.containsSlotValue(wrapper, seen) {
			t.Fatalf("ordinary wrapper %s contains an Owner slot", wrapper)
		}
		if seen[a] || seen[b] || len(seen) > 3 {
			t.Fatalf("wrapper %s expanded the cached cycle: %d types visited", wrapper, len(seen))
		}
	}
}

// TestSlotCacheQueryOrder checks that a cycle does not cache an incomplete negative result.
func TestSlotCacheQueryOrder(t *testing.T) {
	pass := sourcePass(t, `type Model struct{im.Scoped}
 type A struct{ Next []B; Slot im.Owner[Model] }
 type B struct{ Next []A }
 type Empty struct{ Next []Empty }
 type Mixed struct{ Plain []Empty; Next []A }
 type Alias = A
 type Pointer *A
 type Contract interface{ Take(A) }
 type Callback func(A) B`)
	queries := []struct {
		name string
		want bool
	}{
		{"A", true}, {"B", true}, {"Empty", false}, {"Mixed", true}, {"Alias", true},
		{"Pointer", false}, {"Contract", false}, {"Callback", true},
	}
	for start := range queries {
		for _, step := range []int{1, -1} {
			c := &subsetChecks{}
			for offset := range 2 * len(queries) {
				query := queries[(start+step*offset+2*len(queries))%len(queries)]
				typ := pass.Pkg.Scope().Lookup(query.name).Type()
				if got := c.hasSlotValue(typ); got != query.want {
					t.Fatalf("start %d, step %d, %s: got %v, want %v", start, step, query.name, got, query.want)
				}
			}
		}
	}
}

// TestSlotCachePositiveHit checks that a completed positive result is reused inside a fresh wrapper.
func TestSlotCachePositiveHit(t *testing.T) {
	pass := sourcePass(t, `type Model struct{im.Scoped};type Holder struct{Slot im.Owner[Model]}`)
	holder := pass.Pkg.Scope().Lookup("Holder").Type()
	c := &subsetChecks{}
	if !c.hasSlotValue(holder) {
		t.Fatal("holder was not classified")
	}
	seen := make(map[types.Type]bool)
	if !c.containsSlotValue(types.NewSlice(holder), seen) || len(seen) != 1 {
		t.Fatalf("positive cache was not reused: %d types visited", len(seen))
	}
}
