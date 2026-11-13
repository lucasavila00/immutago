package im

import (
	"math/rand/v2"
	"slices"
	"testing"
)

// TestSliceBoundaries checks construction, reads, and append at each trie boundary.
func TestSliceBoundaries(t *testing.T) {
	for _, size := range []int{0, 1, 31, 32, 33, 63, 64, 65, 1023, 1024, 1025, 1056, 1057, 32768, 32800, 32801} {
		input := make([]int, size)
		for index := range input {
			input[index] = index
		}
		original := NewSlice(input...)
		if original.Len() != size || !slices.Equal(original.Values(), input) {
			t.Fatalf("construction failed at length %d", size)
		}
		for index, want := range input {
			if got := original.At(index); got != want {
				t.Fatalf("length %d index %d: got %d, want %d", size, index, got, want)
			}
		}
		grown := original.Append(size, size+1, size+2)
		if grown.Len() != size+3 || grown.At(size+2) != size+2 || original.Len() != size {
			t.Fatalf("append failed at length %d", size)
		}
		if size > 0 {
			changed := original.Set(size/2, -1)
			if changed.At(size/2) != -1 || original.At(size/2) != size/2 {
				t.Fatalf("replacement changed the original at length %d", size)
			}
		}
	}
}

// TestSliceCopies checks that input, output, and update storage cannot mutate existing versions.
func TestSliceCopies(t *testing.T) {
	input := []int{1, 2, 3}
	original := NewSlice(input...)
	input[0] = 99
	output := original.Values()
	output[1] = 99
	left := original.Append(4)
	right := original.Append(5)
	changed := left.Set(0, -1)
	if !slices.Equal(original.Values(), []int{1, 2, 3}) || left.At(3) != 4 || right.At(3) != 5 {
		t.Fatal("a write changed another version")
	}
	if changed.At(0) != -1 || left.At(0) != 1 {
		t.Fatal("Set changed its source")
	}
	if original.Append().root != original.root {
		t.Fatal("empty append changed the tree")
	}
}

// TestSliceRandomVersions compares random changes and old snapshots with native slice copies.
func TestSliceRandomVersions(t *testing.T) {
	random := rand.New(rand.NewPCG(12, 34))
	versions := []Slice[int]{{}}
	models := [][]int{nil}
	for step := 0; step < 2000; step++ {
		parent := random.IntN(len(versions))
		current, model := versions[parent], slices.Clone(models[parent])
		if len(model) != 0 && random.IntN(2) == 0 {
			index, value := random.IntN(len(model)), random.IntN(10000)
			current = current.Set(index, value)
			model[index] = value
		} else {
			values := make([]int, random.IntN(80)+1)
			for index := range values {
				values[index] = random.IntN(10000)
			}
			current = current.Append(values...)
			model = append(model, values...)
		}
		versions = append(versions, current)
		models = append(models, model)
	}
	for index, version := range versions {
		if version.Len() != len(models[index]) || !slices.Equal(version.Values(), models[index]) {
			t.Fatalf("version %d differs from model", index)
		}
	}
}

// TestSliceRepeatedAppend checks tree growth and persistent branches across large boundaries.
func TestSliceRepeatedAppend(t *testing.T) {
	var current Slice[int]
	for index := 0; index < 40000; index++ {
		current = current.Append(index)
	}
	for index := 0; index < current.Len(); index++ {
		if current.At(index) != index {
			t.Fatalf("unexpected value at %d", index)
		}
	}
	changed := current.Set(3, -1)
	if changed.root == current.root || changed.root.children[1] != current.root.children[1] {
		t.Fatal("Set did not copy only the changed path")
	}
	if current.At(3) != 3 || changed.At(3) != -1 {
		t.Fatal("Set corrupted a shared leaf")
	}
}

// TestSliceBounds checks native-style panic behavior for negative and upper-bound indices.
func TestSliceBounds(t *testing.T) {
	for _, size := range []int{0, 1, 32, 33, 1025} {
		s := NewSlice(make([]int, size)...)
		for _, index := range []int{-1, size} {
			expectPanic(t, func() { _ = s.At(index) })
			expectPanic(t, func() { _ = s.Set(index, 1) })
		}
	}
}

// TestSliceReadAllocations verifies that indexed reads allocate no memory.
func TestSliceReadAllocations(t *testing.T) {
	s := NewSlice(make([]int, 40000)...)
	if allocations := testing.AllocsPerRun(100, func() { _ = s.At(30000) }); allocations != 0 {
		t.Fatalf("At allocated %g times", allocations)
	}
}

// expectPanic verifies that an operation rejects invalid input.
func expectPanic(t *testing.T, operation func()) {
	t.Helper()
	defer func() {
		if recover() == nil {
			t.Error("expected a panic")
		}
	}()
	operation()
}
