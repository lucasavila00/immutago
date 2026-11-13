package im

import (
	"maps"
	"slices"
	"testing"
)

// TestSliceAll checks order, repeated traversal, early termination, and empty inputs.
func TestSliceAll(t *testing.T) {
	frozen := FreezeSlice([]int{2, 4, 6})
	iterator := frozen.All()
	for range 2 {
		var got []int
		for index, value := range iterator {
			if index != len(got) {
				t.Fatal("slice iteration order changed")
			}
			got = append(got, value)
		}
		if !slices.Equal(got, []int{2, 4, 6}) {
			t.Fatal("slice iteration contents changed")
		}
	}
	visits := 0
	iterator(func(int, int) bool { visits++; return false })
	if visits != 1 {
		t.Fatal("slice iterator did not stop immediately")
	}
	for range (FrozenSlice[int]{}).All() {
		t.Fatal("zero slice yielded a value")
	}
}

// TestMapAll checks complete traversal, repeated use, early termination, and empty inputs.
func TestMapAll(t *testing.T) {
	input := map[string]int{"a": 1, "b": 2, "c": 3}
	iterator := FreezeMap(input).All()
	for range 2 {
		got := make(map[string]int)
		for key, value := range iterator {
			got[key] = value
		}
		if !maps.Equal(got, input) {
			t.Fatal("map iteration contents changed")
		}
	}
	visits := 0
	iterator(func(string, int) bool { visits++; return false })
	if visits != 1 {
		t.Fatal("map iterator did not stop immediately")
	}
	for range (FrozenMap[string, int]{}).All() {
		t.Fatal("zero map yielded a value")
	}
}

// TestFreezeAndReadsDoNotAllocate checks freeze operations, reads, and direct iteration.
func TestFreezeAndReadsDoNotAllocate(t *testing.T) {
	sequence := []int{1, 2, 3, 4}
	entries := map[int]int{1: 2, 3: 4}
	cases := []struct {
		name string
		run  func()
	}{
		{"freeze slice", func() { benchmarkSlice = FreezeSlice(sequence) }},
		{"freeze map", func() { benchmarkMap = FreezeMap(entries) }},
		{"slice reads", func() { s := FreezeSlice(sequence); benchmarkInt = s.Len() + s.At(1) }},
		{"map reads", func() { m := FreezeMap(entries); benchmarkInt, benchmarkFound = m.Get(1); benchmarkInt += m.Len() }},
		{"slice iteration", func() {
			for index, value := range FreezeSlice(sequence).All() {
				benchmarkInt = index + value
			}
		}},
		{"map iteration", func() {
			for key, value := range FreezeMap(entries).All() {
				benchmarkInt = key + value
			}
		}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if allocations := testing.AllocsPerRun(100, tc.run); allocations != 0 {
				t.Fatalf("got %g allocations; want zero", allocations)
			}
		})
	}
}
