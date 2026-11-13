package borrow

import "testing"

// TestStateJoinKeepsSnapshotsIndependent checks that sibling joins cannot overwrite shared slice capacity.
func TestStateJoinKeepsSnapshotsIndependent(t *testing.T) {
	refs := make([]reference, 3, 4)
	for index := range refs {
		refs[index] = reference{place: Place{Root: index}, loan: -1}
	}
	original := emptyState()
	original.cells["slot"] = cell{place: Place{Root: 9}, refs: refs}
	left, right := original.clone(), original.clone()
	leftInput, rightInput := emptyState(), emptyState()
	leftInput.cells["slot"] = cell{place: Place{Root: 9}, refs: []reference{{place: Place{Root: 3}, loan: -1}}}
	rightInput.cells["slot"] = cell{place: Place{Root: 9}, refs: []reference{{place: Place{Root: 4}, loan: -1}}}
	left.join(leftInput)
	right.join(rightInput)
	if len(original.cells["slot"].refs) != 3 {
		t.Fatal("join changed the source snapshot")
	}
	if left.cells["slot"].refs[3].place.Root != 3 || right.cells["slot"].refs[3].place.Root != 4 {
		t.Fatal("sibling joins share mutable reference storage")
	}
}

// TestFreshAliasSummaryIdentity checks that fresh result groups and unknown provenance remain distinct facts.
func TestFreshAliasSummaryIdentity(t *testing.T) {
	original := Result{Fresh: true, FreshGroup: "0,1,", Protected: true}
	for _, changed := range []Result{
		{Fresh: true, FreshGroup: "0,", Protected: true},
		{Fresh: true, FreshGroup: "0,1,", Protected: true, Shared: true},
		{Fresh: true, FreshGroup: "0,1,", Protected: true, Unknown: true},
		{Fresh: true, FreshGroup: "0,1,", Protected: true, Global: true},
	} {
		if sameResult(original, changed) {
			t.Fatalf("merged distinct effects: %+v", changed)
		}
	}
}
