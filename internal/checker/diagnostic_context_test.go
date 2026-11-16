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
		for _, payload := range []types.Type{marker, alias, types.NewArray(alias, 0)} {
			checkMarkerPayload(t, name, payload)
		}
	}
}

// checkMarkerPayload checks one annotation through every proof category.
func checkMarkerPayload(t *testing.T, name string, payload types.Type) {
	t.Helper()
	var prover Prover
	for _, prove := range []func(types.Type) Failure{
		prover.FrozenFailure, prover.CopiedFailure, prover.ReadResultFailure,
	} {
		failure := prove(payload)
		if !strings.Contains(failure.Reason, "immutable markers are annotations, not data") {
			t.Errorf("%s: %v", payload, failure)
		}
		want := "embed the marker in a named struct"
		if name == "FrozenInterface" {
			want = "embed im.FrozenInterface in a named interface"
		}
		if !strings.Contains(failure.Repair, want) {
			t.Errorf("%s: %v", payload, failure)
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
