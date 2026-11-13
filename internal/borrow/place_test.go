package borrow

import "testing"

// TestPlaceConflictBias checks the rustc may-overlap versus definite-overlap distinction.
func TestPlaceConflictBias(t *testing.T) {
	root := Place{Root: 1}
	field := root.project('f', 0)
	cases := []struct {
		name        string
		left, right Place
		may, must   bool
	}{
		{"different allocations", root, Place{Root: 2}, false, false},
		{"same allocation", root, root, true, true},
		{"parent and child", root, field, true, true},
		{"different fields", field, root.project('f', 1), false, false},
		{"same index", field.project('i', 1), field.project('i', 1), true, true},
		{"different indices", field.project('i', 0), field.project('i', 1), false, false},
		{"unknown index", field.project('?', 0), field.project('i', 1), true, false},
		{"different subfields", root.project('?', 0).project('f', 0), root.project('?', 0).project('f', 1), false, false},
	}
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			if got := Overlap(test.left, test.right, false); got != test.may {
				t.Fatalf("possible=%v want=%v", got, test.may)
			}
			if got := Overlap(test.left, test.right, true); got != test.must {
				t.Fatalf("definite=%v want=%v", got, test.must)
			}
		})
	}
}

// TestStateJoinPreservesSiblingStates checks that a join cannot mutate a sibling's alias-set backing array.
func TestStateJoinPreservesSiblingStates(t *testing.T) {
	refs := make([]reference, 3, 4)
	for i := range refs {
		refs[i] = reference{place: Place{Root: i}, loan: -1}
	}
	left := append([]reference(nil), refs...)
	joined, _ := mergeReferences(refs, []reference{{place: Place{Root: 3}, loan: -1}})
	other, _ := mergeReferences(refs, []reference{{place: Place{Root: 4}, loan: -1}})
	if joined[3].place.Root != 3 || other[3].place.Root != 4 || !sameReferences(refs, left) {
		t.Fatal("a sibling join overwrote shared alias storage")
	}
}
