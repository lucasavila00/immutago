package checker

import (
	"go/token"
	"go/types"
	"strings"
	"testing"
)

// TestFrozenNamedFields checks imported structures without source declarations.
func TestFrozenNamedFields(t *testing.T) {
	marker, _ := testTypes()
	pkg := types.NewPackage("example.test/foreign", "foreign")
	alias := types.NewAlias(types.NewTypeName(token.NoPos, pkg, "Marker", nil), marker)
	count := types.NewNamed(types.NewTypeName(token.NoPos, pkg, "Count", nil), types.Typ[types.Int], nil)
	child := markedType("Child", marker)
	for _, data := range []types.Type{count, child, types.NewPointer(child)} {
		for _, embedded := range []bool{false, true} {
			fields := []*types.Var{
				types.NewField(token.NoPos, pkg, "Marker", alias, true),
				types.NewField(token.NoPos, pkg, "Data", data, embedded),
			}
			value := types.NewNamed(types.NewTypeName(token.NoPos, pkg, "Record", nil), types.NewStruct(fields, nil), nil)
			pointer := types.NewPointer(value)
			prover := new(Prover)
			for _, root := range []types.Type{pointer, value, types.NewArray(pointer, 2)} {
				reason := prover.FrozenReason(root)
				if embedded && !strings.Contains(reason, "Data: Frozen data fields must be named") {
					t.Fatalf("embedded %s accepted: %s", data, reason)
				}
				if !embedded && reason != "" {
					t.Fatalf("named %s rejected: %s", data, reason)
				}
			}
		}
	}
}
