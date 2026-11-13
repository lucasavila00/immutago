package im

import (
	"slices"
	"strconv"
	"testing"
)

// TestFreezeSlice checks zero-copy storage and bounds for empty and nonempty slices.
func TestFreezeSlice(t *testing.T) {
	input := []int{3, 5, 8}
	frozen := FreezeSlice(input)
	if frozen.Len() != len(input) || frozen.At(1) != 5 {
		t.Fatal("slice contents changed")
	}
	if &frozen.values[0] != &input[0] {
		t.Fatal("freeze copied the backing array")
	}
	for _, index := range []int{-1, 3} {
		t.Run(strconv.Itoa(index), func(t *testing.T) {
			defer func() {
				if recover() == nil {
					t.Fatal("out-of-range access did not panic")
				}
			}()
			frozen.At(index)
		})
	}
}

// TestSliceClone checks independent copies, including a nonzero-offset view of larger storage.
func TestSliceClone(t *testing.T) {
	input := []int{1, 2, 3, 4, 5}
	frozen := FreezeSlice(input[1:4])
	first, second := frozen.Clone(), frozen.Clone()
	if !slices.Equal(first, []int{2, 3, 4}) {
		t.Fatal("clone contents changed")
	}
	first[0] = 99
	first = append(first, 6)
	if frozen.At(0) != 2 || !slices.Equal(second, []int{2, 3, 4}) || len(first) != 4 {
		t.Fatal("clone shares mutable storage")
	}
}

// TestSliceZeroValues checks nil preservation and empty read operations.
func TestSliceZeroValues(t *testing.T) {
	var zero FrozenSlice[int]
	if zero.Len() != 0 || zero.Clone() != nil || FreezeSlice[int](nil).Clone() != nil {
		t.Fatal("invalid zero slice")
	}
	if empty := FreezeSlice([]int{}).Clone(); empty == nil || len(empty) != 0 {
		t.Fatal("empty clone lost nil distinction")
	}
	defer func() {
		if recover() == nil {
			t.Fatal("empty indexing did not panic")
		}
	}()
	zero.At(0)
}
