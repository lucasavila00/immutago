package checker

import (
	"go/token"
	"go/types"
	"strings"
	"testing"
)

// frozenContract constructs compiler identities for one sealed test interface.
func frozenContract(name string, methods ...*types.Func) *types.Named {
	library := types.NewPackage(LibraryPath, "im")
	seal := types.NewFunc(token.NoPos, library, "immutagoFrozen", types.NewSignatureType(nil, nil, nil, nil, nil, false))
	contract := types.NewInterfaceType(append([]*types.Func{seal}, methods...), nil).Complete()
	pkg := types.NewPackage("example.com/data", "data")
	return types.NewNamed(types.NewTypeName(token.NoPos, pkg, name, nil), contract, nil)
}

// TestFrozenInterfaceCategories checks sealing, result proofs, aliases, and category separation.
func TestFrozenInterfaceCategories(t *testing.T) {
	method := func(name string, result ...types.Type) *types.Func {
		var variables []*types.Var
		for _, typ := range result {
			variables = append(variables, types.NewVar(token.NoPos, nil, "", typ))
		}
		results := types.NewTuple(variables...)
		return types.NewFunc(token.NoPos, nil, name, types.NewSignatureType(nil, nil, nil, nil, results, false))
	}
	good := frozenContract("Scorer", method("Score", types.Typ[types.Float64]),
		method("Lookup", types.Typ[types.Float64], types.Typ[types.Bool], types.NewSlice(types.Typ[types.Byte]),
			types.Universe.Lookup("error").Type()))
	bad := frozenContract("Text", method("Text", types.Typ[types.String]))
	alias := types.NewAlias(types.NewTypeName(token.NoPos, nil, "Alias", nil), good)
	for _, typ := range []types.Type{good, alias} {
		var proofs Prover
		var queries Classifier
		if !FrozenInterface(typ) || !queries.Contains(typ) ||
			queries.Locals(typ) || queries.Callable(typ) {
			t.Errorf("incorrect interface classification: %s", typ)
		}
		if reason := proofs.FrozenReason(typ); reason != "" {
			t.Errorf("Frozen interface failed: %s", reason)
		}
		if proofs.CopiedReason(typ) == "" || proofs.ReadResultReason(typ) == "" {
			t.Error("Frozen interface escaped through a read result")
		}
	}
	parameter := types.NewTypeParam(types.NewTypeName(token.NoPos, nil, "T", nil), good)
	var queries Classifier
	if !queries.Contains(parameter) {
		t.Fatal("a Frozen interface constraint lost its protected type")
	}
	var proofs Prover
	if reason := proofs.FrozenReason(parameter); reason == "" {
		t.Fatal("a type parameter passed the concrete interface proof")
	}
	if reason := proofs.FrozenReason(bad); !strings.Contains(reason, "Text result:") {
		t.Fatalf("invalid result was accepted: %s", reason)
	}
	if reason := proofs.FrozenReason(good.Underlying()); !strings.Contains(reason, "named non-generic") {
		t.Fatalf("anonymous interface was accepted: %s", reason)
	}
	foreignSeal := types.NewFunc(token.NoPos, types.NewPackage("other/im", "im"), "immutagoFrozen",
		types.NewSignatureType(nil, nil, nil, nil, nil, false))
	if FrozenInterface(types.NewInterfaceType([]*types.Func{foreignSeal}, nil).Complete()) {
		t.Fatal("a different package supplied the seal")
	}
}
