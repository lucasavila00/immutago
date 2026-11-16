package analyzer

import (
	"go/types"

	"github.com/lucasavila00/immutago/internal/checker"
)

// valueFailure proves Value's payload and gives a replacement for its projection input.
func valueFailure(proofs *checker.Prover, payload types.Type) checker.Failure {
	if failure := proofs.FrozenFailure(payload); failure.Reason != "" {
		return failure
	}
	failure := proofs.ReadResultFailure(payload)
	if failure.Reason == "" {
		return failure
	}
	switch {
	case stringType(payload):
		failure.Repair = "use im.StringOf(projection), then Unsafe for shared text or Clone for an independent copy"
	case checker.IsLibraryType(payload, "FrozenString"):
		failure.Repair = "use im.StringOfFrozen(projection), then Unsafe for shared text or Clone for an independent copy"
	case checker.IsLibraryType(payload, "FrozenSlice"):
		failure.Repair = "use im.SliceOf(projection) for element reads, or projection.Unsafe() for the shared FrozenSlice"
	case checker.IsLibraryType(payload, "FrozenMap"):
		failure.Repair = "use im.MapOf(projection) for entry reads, or projection.Unsafe() for the shared FrozenMap"
	case checker.Frozen(payload):
		failure.Repair = "use projection.Unsafe() for the shared record; for Local field reads, pass " +
			"im.CopyRecord(projection) to its generated projection constructor (copies the root)"
	default:
		failure.Repair = "use projection.Unsafe() for shared access; use the matching generated projection " +
			"for Local reads when available"
	}
	return failure
}
