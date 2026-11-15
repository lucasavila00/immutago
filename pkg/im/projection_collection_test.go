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

// TestSliceProjection checks shared storage, element projections, and independent native copies.
func TestSliceProjection(t *testing.T) {
	input := []int{2, 4, 6}
	projection := Slice(ProjectionOf(FreezeSlice(input)))
	if &projection.values[0] != &input[0] || projection.Len() != 3 || Copy(projection.At(1)) != 4 {
		t.Fatal("slice adapter changed storage or contents")
	}
	first, second := projection.Clone(), projection.Clone()
	first[0] = 9
	if Copy(projection.At(0)) != 2 || !slices.Equal(second, input) {
		t.Fatal("slice clone shares storage")
	}
	text := strings.Repeat("text", 128)
	texts := Slice(ProjectionOf(FreezeSlice([]string{text})))
	if unsafe.StringData(texts.At(0).value) != unsafe.StringData(text) {
		t.Fatal("element projection copied its backing storage")
	}
}

// TestSliceProjectionEmpty checks nil preservation and empty clones that release backing storage.
func TestSliceProjectionEmpty(t *testing.T) {
	var zero SliceProjection[int]
	if zero.Len() != 0 || zero.Clone() != nil {
		t.Fatal("invalid zero slice projection")
	}
	input := make([]int, 1024)
	projection := Slice(ProjectionOf(FreezeSlice(input[512:512])))
	clone := projection.Clone()
	if clone == nil || len(clone) != 0 || cap(clone) != 0 || unsafe.SliceData(clone) == unsafe.SliceData(projection.values) {
		t.Fatal("empty clone retained backing storage or lost nil distinction")
	}
	for _, current := range []SliceProjection[int]{zero, Slice(ProjectionOf(FreezeSlice([]int{1})))} {
		for _, index := range []int{-1, current.Len()} {
			func() {
				defer func() {
					if recover() == nil {
						t.Error("out-of-range slice projection read did not panic")
					}
				}()
				_ = current.At(index)
			}()
		}
	}
}

// TestMapProjection checks shared storage, value projections, and independent native copies.
func TestMapProjection(t *testing.T) {
	input := map[int]int{1: 2, 3: 4}
	projection := Map(ProjectionOf(FreezeMap(input)))
	if reflect.ValueOf(projection.values).UnsafePointer() != reflect.ValueOf(input).UnsafePointer() ||
		projection.Len() != 2 {
		t.Fatal("map adapter changed storage")
	}
	if value, found := projection.Get(3); !found || Copy(value) != 4 {
		t.Fatal("map projection changed contents")
	}
	if value, found := projection.Get(9); found || Copy(value) != 0 {
		t.Fatal("missing key did not return the zero projection and false")
	}
	first, second := projection.Clone(), projection.Clone()
	first[1] = 9
	delete(first, 3)
	if !maps.Equal(second, input) || projection.Len() != 2 {
		t.Fatal("map clone shares storage")
	}
	var zero MapProjection[int, int]
	if zero.Len() != 0 || zero.Clone() != nil {
		t.Fatal("invalid zero map projection")
	}
	empty := Map(ProjectionOf(FreezeMap(map[int]int{})))
	clone := empty.Clone()
	if clone == nil {
		t.Fatal("empty clone lost nil distinction")
	}
	clone[1] = 2
	if empty.Len() != 0 {
		t.Fatal("empty map clone shares storage")
	}
}

// TestMapProjectionNested checks that map reads wrap string keys and pointer values.
func TestMapProjectionNested(t *testing.T) {
	key := strings.Repeat("key", 128)
	child := &testChild{Number: 7}
	projection := Map(ProjectionOf(FreezeMap(map[string]*testChild{key: child})))
	if value, found := projection.Get(key); !found || value.value != child {
		t.Fatal("map value projection changed pointer identity")
	}
	if value, found := projection.Get("missing"); found || !IsNil(value) {
		t.Fatal("missing pointer value is not nil")
	}
	for k, v := range projection.All() {
		if unsafe.StringData(k.value) != unsafe.StringData(key) || v.value != child {
			t.Fatal("map iteration copied key or value backing storage")
		}
	}
}

// TestMapProjectionKeySemantics checks NaN entries and signed-zero lookup.
func TestMapProjectionKeySemantics(t *testing.T) {
	nan := math.NaN()
	input := map[float64]int{nan: 1}
	input[nan] = 2
	input[math.Copysign(0, -1)] = 3
	projection := Map(ProjectionOf(FreezeMap(input)))
	if _, found := projection.Get(nan); found {
		t.Fatal("NaN unexpectedly compares equal")
	}
	if value, found := projection.Get(0); !found || Copy(value) != 3 {
		t.Fatal("signed zero lookup differs")
	}
	if projection.Len() != 3 || len(projection.Clone()) != 3 {
		t.Fatal("map clone lost NaN entries")
	}
}

// TestProjectionIteration checks order, repeat use, early stop, and empty projections.
func TestProjectionIteration(t *testing.T) {
	sequence := Slice(ProjectionOf(FreezeSlice([]int{2, 4, 6})))
	mapping := Map(ProjectionOf(FreezeMap(map[int]int{1: 2, 3: 4})))
	for range 2 {
		var got []int
		for index, value := range sequence.All() {
			if index != len(got) {
				t.Fatal("slice projection iteration order changed")
			}
			got = append(got, Copy(value))
		}
		if !slices.Equal(got, sequence.values) {
			t.Fatal("slice projection iteration changed contents")
		}
		entries := make(map[int]int)
		for key, value := range mapping.All() {
			entries[Copy(key)] = Copy(value)
		}
		if !maps.Equal(entries, mapping.values) {
			t.Fatal("map projection iteration changed contents")
		}
	}
	visits := 0
	for range sequence.All() {
		visits++
		break
	}
	for range mapping.All() {
		visits++
		break
	}
	if visits != 2 {
		t.Fatal("projection iterator did not stop immediately")
	}
	for range (SliceProjection[int]{}).All() {
		t.Fatal("empty slice projection yielded an entry")
	}
	for range (MapProjection[int, int]{}).All() {
		t.Fatal("empty map projection yielded an entry")
	}
}
