package borrow

import "testing"

// TestMergeSummaryPreservesUnknown prevents wrappers from hiding unavailable callee effects.
func TestMergeSummaryPreservesUnknown(t *testing.T) {
	left := Summary{Inputs: []Effect{{Read: true}}}
	right := Summary{Inputs: []Effect{{Unknown: true, Capture: true}}}
	merged, changed := mergeSummary(left, right)
	if !changed || !merged.Inputs[0].Read || !merged.Inputs[0].Unknown || !merged.Inputs[0].Capture {
		t.Fatalf("lost input effects: %+v", merged)
	}
	if _, changed := mergeSummary(merged, right); changed {
		t.Fatal("stable effects changed")
	}
}

// TestRecursiveReturnPathsConverge checks finite widening without an arbitrary iteration limit.
func TestRecursiveReturnPathsConverge(t *testing.T) {
	first := Result{Input: 0, Shared: true, Path: []Projection{{Kind: 'f', Index: 0}}}
	summary := Summary{Results: [][]Result{{first}}}
	deeper := first
	deeper.Path = append(append([]Projection(nil), first.Path...), Projection{Kind: 'f', Index: 1})
	merged, changed := mergeSummary(summary, Summary{Results: [][]Result{{deeper}}})
	if changed || len(merged.Results[0]) != 1 || len(merged.Results[0][0].Path) != 1 {
		t.Fatalf("longer recursive alias did not converge: %+v", merged.Results)
	}
	other := first
	other.Path = []Projection{{Kind: 'f', Index: 1}}
	merged, changed = mergeSummary(summary, Summary{Results: [][]Result{{other}}})
	if !changed || len(merged.Results[0][0].Path) != 0 {
		t.Fatal("did not conservatively join differing fields")
	}
	if len(summary.Results[0][0].Path) != 1 {
		t.Fatal("modified the previous summary")
	}
}

// TestFreshReturnGroupsPreserveAliasShape keeps same-object and independent result groups distinct.
func TestFreshReturnGroupsPreserveAliasShape(t *testing.T) {
	same := Result{Input: -1, Fresh: true, FreshGroup: "0,1", Protected: true}
	separate := Result{Input: -1, Fresh: true, FreshGroup: "0", Protected: true}
	summary := Summary{Results: [][]Result{{same}, {same}}}
	merged, changed := mergeSummary(summary, Summary{Results: [][]Result{{separate}}})
	if !changed || len(merged.Results[0]) != 2 || len(merged.Results[1]) != 1 {
		t.Fatal("collapsed distinct result alias shapes")
	}
	if _, changed := mergeSummary(merged, summary); changed {
		t.Fatal("stable fresh aliases changed")
	}
	global := same
	global.Global = true
	global.Fresh = false
	merged, changed = mergeSummary(merged, Summary{Results: [][]Result{{global}}})
	if !changed || len(merged.Results[0]) != 3 {
		t.Fatal("lost a global returned alias")
	}
}
