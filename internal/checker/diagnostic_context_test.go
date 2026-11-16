package checker

import (
	"go/token"
	"go/types"
	"strings"
	"testing"
)

// TestMarkerPayloadFailures keeps all annotation payloads invalid in every category.
func TestMarkerPayloadFailures(t *testing.T) {
	frozen, _ := testTypes()
	pkg := frozen.Obj().Pkg()
	for _, name := range []string{"Frozen", "Local", "Copied", "FrozenInterface"} {
		marker := types.NewNamed(types.NewTypeName(token.NoPos, pkg, name, nil), types.NewStruct(nil, nil), nil)
		if name == "FrozenInterface" {
			marker.SetUnderlying(frozenContract("Marker").Underlying())
		}
		alias := types.NewAlias(types.NewTypeName(token.NoPos, nil, "Alias", nil), marker)
		for _, payload := range []types.Type{
			marker, alias, types.NewArray(alias, 0), types.NewArray(types.NewArray(alias, 2), 3),
		} {
			checkMarkerPayload(t, name, payload)
		}
	}
}

// checkMarkerPayload checks one annotation through every proof category.
func checkMarkerPayload(t *testing.T, name string, payload types.Type) {
	t.Helper()
	want := [3]string{}
	for category := range want {
		want[category] = "embed the marker in a named struct; use data fields for values"
	}
	if name == "FrozenInterface" {
		want = [3]string{
			"embed im.FrozenInterface in a named interface; use that interface as the payload",
			"use bool, numbers, unnamed arrays, or Copied records; " +
				"for Local fields or results, use matching projections; " +
				"if a Copied record needs text, opaque handles, or Local fields, " +
				"replace its im.Copied marker with im.Local and use read-result fields such as im.StringProjection",
			"embed im.FrozenInterface in a named interface; use its generated projection as the read result",
		}
	}
	for _, order := range [][3]int{{0, 1, 2}, {2, 1, 0}, {1, 0, 2}} {
		var prover Prover
		checks := []func(types.Type) Failure{prover.FrozenFailure, prover.CopiedFailure, prover.ReadResultFailure}
		for range 2 {
			for _, category := range order {
				failure := checks[category](payload)
				if !strings.Contains(failure.Reason, "immutable markers are annotations, not data") ||
					failure.Repair != want[category] {
					t.Errorf("%s, category %d, order %v: %v; want repair %q", payload, category, order, failure, want[category])
				}
			}
		}
	}
}

// TestLocalFailurePaths keeps the field and requested category through nested owners and cached queries.
func TestLocalFailurePaths(t *testing.T) {
	frozen, slice := testTypes()
	pkg := frozen.Obj().Pkg()
	local := types.NewNamed(types.NewTypeName(token.NoPos, pkg, "Local", nil), types.NewStruct(nil, nil), nil)
	copied := types.NewNamed(types.NewTypeName(token.NoPos, pkg, "Copied", nil), types.NewStruct(nil, nil), nil)
	text := types.NewNamed(types.NewTypeName(token.NoPos, pkg, "StringProjection", nil), types.NewStruct(nil, nil), nil)
	alias := types.NewAlias(types.NewTypeName(token.NoPos, nil, "Text", nil), text)
	reply := markedType("Reply", local, types.NewField(token.NoPos, nil, "Name", text, false))
	for _, leaf := range []types.Type{text, alias, reply, types.NewArray(alias, 0), types.NewArray(reply, 2)} {
		field := types.NewField(token.NoPos, nil, "Label", leaf, false)
		score := markedType("Score", copied, field)
		outer := markedType("Outer", copied, types.NewField(token.NoPos, nil, "Score", score, false))
		settings := markedType("Settings", frozen, field)
		for _, first := range []types.Type{leaf, score, outer, settings} {
			checkLocalFailureQuery(t, first, score, outer, settings, slice)
		}
	}
}

// checkLocalFailureQuery checks both owner categories after a different initial query.
func checkLocalFailureQuery(t *testing.T, first, score, outer, settings types.Type, slice *types.Named) {
	t.Helper()
	var prover Prover
	_ = prover.FrozenFailure(first)
	_ = prover.ReadResultFailure(first)
	for _, owner := range []types.Type{score, outer} {
		failure := prover.CopiedFailure(owner)
		if !strings.Contains(failure.Reason, "Label: ") ||
			!strings.Contains(failure.Reason, "is not a Copied value") ||
			!strings.Contains(failure.Repair, "replace its im.Copied marker with im.Local") {
			t.Errorf("%s after %s: %v", owner, first, failure)
		}
		if again := prover.CopiedFailure(owner); again != failure {
			t.Errorf("cached failure changed: %v -> %v", failure, again)
		}
	}
	for _, owner := range []types.Type{settings, types.NewPointer(settings), sliceType(t, slice, settings)} {
		if failure := prover.FrozenFailure(owner); !strings.Contains(failure.Reason,
			"Label: Frozen storage cannot contain Local values") {
			t.Errorf("%s after %s: %v", owner, first, failure)
		}
	}
}

// TestReadResultFailurePaths keeps result failures separate from Copied field failures in either query order.
func TestReadResultFailurePaths(t *testing.T) {
	frozen, _ := testTypes()
	pkg := frozen.Obj().Pkg()
	copied := types.NewNamed(types.NewTypeName(token.NoPos, pkg, "Copied", nil), types.NewStruct(nil, nil), nil)
	local := types.NewNamed(types.NewTypeName(token.NoPos, pkg, "Local", nil), types.NewStruct(nil, nil), nil)
	worker := types.NewNamed(types.NewTypeName(token.NoPos, nil, "Worker", nil), types.NewStruct(nil, nil), nil)
	pointer := types.NewPointer(worker)
	alias := types.NewAlias(types.NewTypeName(token.NoPos, nil, "WorkerPointer", nil), pointer)
	text := types.NewField(token.NoPos, nil, "Text", types.Typ[types.String], false)
	cases := []struct {
		typ    types.Type
		reason string
		repair string
	}{
		{pointer, "*Worker is not a read result", "an opaque handle"},
		{alias, "*Worker is not a read result", "an opaque handle"},
		{types.NewArray(types.NewArray(alias, 2), 0),
			"array element: array element: *Worker is not a read result", "an opaque handle"},
		{markedType("Reply", local, types.NewField(token.NoPos, nil, "Worker", pointer, false)),
			"Worker: *Worker is not a read result", "an opaque handle"},
		{markedType("Score", copied, text),
			"Text: string is not a Copied value", "replace its im.Copied marker with im.Local"},
		{types.NewNamed(types.NewTypeName(token.NoPos, nil, "Counts", nil), types.NewArray(types.Typ[types.Int], 2), nil),
			"named array must be wrapped", "use an unnamed array type"},
	}
	for _, test := range cases {
		for _, copiedFirst := range []bool{false, true} {
			var prover Prover
			if copiedFirst {
				_ = prover.CopiedFailure(test.typ)
			}
			failure := prover.ReadResultFailure(test.typ)
			if !strings.Contains(failure.Reason, test.reason) || !strings.Contains(failure.Repair, test.repair) {
				t.Errorf("%s, Copied first %t: %v", test.typ, copiedFirst, failure)
			}
			_ = prover.CopiedFailure(test.typ)
			if again := prover.ReadResultFailure(test.typ); again != failure {
				t.Errorf("cached failure changed: %v -> %v", failure, again)
			}
		}
	}
}
