package im

import (
	"maps"
	"math"
	"reflect"
	"slices"
	"strings"
	"testing"
	"unsafe"
)

// TestByValueSlice checks shared storage, reads, independent clones, and nil preservation.
func TestByValueSlice(t *testing.T) {
	input := []int{2, 4, 6}
	frozen := FreezeSlice(input)
	for _, view := range []FrozenByValueSlice[int]{frozen.ByValue(), FreezeByValueSlice(input)} {
		if &view.values[0] != &input[0] || view.Len() != 3 || view.At(1) != 4 {
			t.Fatal("slice view changed storage or contents")
		}
		first, second := view.Clone(), view.Clone()
		first[0] = 9
		if view.At(0) != 2 || !slices.Equal(second, input) {
			t.Fatal("slice clone shares storage")
		}
	}
	var zero FrozenByValueSlice[int]
	if zero.Len() != 0 || zero.Clone() != nil || FreezeByValueSlice[int](nil).Clone() != nil {
		t.Fatal("invalid zero slice view")
	}
	if FreezeByValueSlice([]int{}).Clone() == nil {
		t.Fatal("empty slice clone lost nil distinction")
	}
	for _, index := range []int{-1, 0} {
		func() {
			defer func() {
				if recover() == nil {
					t.Error("out-of-range slice view access did not panic")
				}
			}()
			_ = zero.At(index)
		}()
	}
}

// TestByValueMap checks shared storage, reads, independent clones, and nil preservation.
func TestByValueMap(t *testing.T) {
	input := map[int]int{1: 2, 3: 4}
	frozen := FreezeMap(input)
	for _, view := range []FrozenByValueMap[int, int]{frozen.ByValue(), FreezeByValueMap(input)} {
		if reflect.ValueOf(view.values).UnsafePointer() != reflect.ValueOf(input).UnsafePointer() || view.Len() != 2 {
			t.Fatal("map view changed storage")
		}
		if value, found := view.Get(3); !found || value != 4 {
			t.Fatal("map view changed contents")
		}
		if value, found := view.Get(9); found || value != 0 {
			t.Fatal("missing key did not return zero,false")
		}
		first, second := view.Clone(), view.Clone()
		first[1] = 9
		delete(first, 3)
		if !maps.Equal(second, input) || view.Len() != 2 {
			t.Fatal("map clone shares storage")
		}
	}
	var zero FrozenByValueMap[int, int]
	if zero.Len() != 0 || zero.Clone() != nil || FreezeByValueMap[int, int](nil).Clone() != nil {
		t.Fatal("invalid zero map view")
	}
	if FreezeByValueMap(map[int]int{}).Clone() == nil {
		t.Fatal("empty map clone lost nil distinction")
	}
}

// TestByValueMapStringKeys checks that Clone copies the bytes behind an exact FrozenString alias.
func TestByValueMapStringKeys(t *testing.T) {
	type key = FrozenString
	parent := strings.Repeat("abcdef", 1024)
	original := FreezeString(parent[100:107])
	view := FreezeByValueMap(map[key]int{original: 4})
	first, second := view.Clone(), view.Clone()
	for copied, value := range first {
		if copied != original || value != 4 || unsafe.StringData(copied.value) == unsafe.StringData(original.value) {
			t.Fatal("map clone changed the key or retained its bytes")
		}
		for other := range second {
			if unsafe.StringData(copied.value) == unsafe.StringData(other.value) {
				t.Fatal("map clones share key bytes")
			}
		}
	}
	first[original] = 9
	if value, _ := view.Get(original); value != 4 || second[original] != 4 {
		t.Fatal("map clone changed source values")
	}
}

// TestByValueMapKeySemantics preserves NaN entries and signed-zero lookup.
func TestByValueMapKeySemantics(t *testing.T) {
	nan := math.NaN()
	input := map[float64]int{nan: 1}
	input[nan] = 2
	input[math.Copysign(0, -1)] = 3
	view := FreezeByValueMap(input)
	if _, found := view.Get(nan); found {
		t.Fatal("NaN unexpectedly compares equal")
	}
	if value, found := view.Get(0); !found || value != 3 {
		t.Fatal("signed zero lookup differs")
	}
	if view.Len() != 3 || len(view.Clone()) != 3 {
		t.Fatal("map clone lost NaN entries")
	}
}

// TestByValueIteration checks order, contents, repeat use, early stop, and empty views.
func TestByValueIteration(t *testing.T) {
	sequence := FreezeByValueSlice([]int{2, 4, 6})
	mapping := FreezeByValueMap(map[int]int{1: 2, 3: 4})
	for range 2 {
		var got []int
		for index, value := range sequence.All() {
			if index != len(got) {
				t.Fatal("slice view iteration order changed")
			}
			got = append(got, value)
		}
		if !slices.Equal(got, sequence.values) {
			t.Fatal("slice view iteration changed contents")
		}
		entries := make(map[int]int)
		for key, value := range mapping.All() {
			entries[key] = value
		}
		if !maps.Equal(entries, mapping.values) {
			t.Fatal("map view iteration changed contents")
		}
	}
	visits := 0
	sequence.All()(func(int, int) bool { visits++; return false })
	mapping.All()(func(int, int) bool { visits++; return false })
	if visits != 2 {
		t.Fatal("view iterator did not stop immediately")
	}
	for range (FrozenByValueSlice[int]{}).All() {
		t.Fatal("empty slice view yielded an entry")
	}
	for range (FrozenByValueMap[int, int]{}).All() {
		t.Fatal("empty map view yielded an entry")
	}
}
