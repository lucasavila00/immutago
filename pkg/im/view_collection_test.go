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

// TestSliceView checks shared storage, element views, and independent native copies.
func TestSliceView(t *testing.T) {
	input := []int{2, 4, 6}
	view := Slice(ViewOf(FreezeSlice(input)))
	if &view.values[0] != &input[0] || view.Len() != 3 || Copy(view.At(1)) != 4 {
		t.Fatal("slice adapter changed storage or contents")
	}
	first, second := view.Clone(), view.Clone()
	first[0] = 9
	if Copy(view.At(0)) != 2 || !slices.Equal(second, input) {
		t.Fatal("slice clone shares storage")
	}
	text := strings.Repeat("text", 128)
	texts := Slice(ViewOf(FreezeSlice([]string{text})))
	if unsafe.StringData(texts.At(0).value) != unsafe.StringData(text) {
		t.Fatal("element view copied its backing storage")
	}
}

// TestSliceViewEmpty checks nil preservation and empty clones that release backing storage.
func TestSliceViewEmpty(t *testing.T) {
	var zero SliceView[int]
	if zero.Len() != 0 || zero.Clone() != nil {
		t.Fatal("invalid zero slice view")
	}
	input := make([]int, 1024)
	view := Slice(ViewOf(FreezeSlice(input[512:512])))
	clone := view.Clone()
	if clone == nil || len(clone) != 0 || cap(clone) != 0 || unsafe.SliceData(clone) == unsafe.SliceData(view.values) {
		t.Fatal("empty clone retained backing storage or lost nil distinction")
	}
	for _, current := range []SliceView[int]{zero, Slice(ViewOf(FreezeSlice([]int{1})))} {
		for _, index := range []int{-1, current.Len()} {
			func() {
				defer func() {
					if recover() == nil {
						t.Error("out-of-range slice view read did not panic")
					}
				}()
				_ = current.At(index)
			}()
		}
	}
}

// TestMapView checks shared storage, value views, and independent native copies.
func TestMapView(t *testing.T) {
	input := map[int]int{1: 2, 3: 4}
	view := Map(ViewOf(FreezeMap(input)))
	if reflect.ValueOf(view.values).UnsafePointer() != reflect.ValueOf(input).UnsafePointer() || view.Len() != 2 {
		t.Fatal("map adapter changed storage")
	}
	if value, found := view.Get(3); !found || Copy(value) != 4 {
		t.Fatal("map view changed contents")
	}
	if value, found := view.Get(9); found || Copy(value) != 0 {
		t.Fatal("missing key did not return the zero view and false")
	}
	first, second := view.Clone(), view.Clone()
	first[1] = 9
	delete(first, 3)
	if !maps.Equal(second, input) || view.Len() != 2 {
		t.Fatal("map clone shares storage")
	}
	var zero MapView[int, int]
	if zero.Len() != 0 || zero.Clone() != nil {
		t.Fatal("invalid zero map view")
	}
	empty := Map(ViewOf(FreezeMap(map[int]int{})))
	clone := empty.Clone()
	if clone == nil {
		t.Fatal("empty clone lost nil distinction")
	}
	clone[1] = 2
	if empty.Len() != 0 {
		t.Fatal("empty map clone shares storage")
	}
}

// TestMapViewNested checks that map reads wrap string keys and pointer values.
func TestMapViewNested(t *testing.T) {
	key := strings.Repeat("key", 128)
	child := &testChild{Number: 7}
	view := Map(ViewOf(FreezeMap(map[string]*testChild{key: child})))
	if value, found := view.Get(key); !found || value.value != child {
		t.Fatal("map value view changed pointer identity")
	}
	if value, found := view.Get("missing"); found || !IsNil(value) {
		t.Fatal("missing pointer value is not nil")
	}
	for k, v := range view.All() {
		if unsafe.StringData(k.value) != unsafe.StringData(key) || v.value != child {
			t.Fatal("map iteration copied key or value backing storage")
		}
	}
}

// TestMapViewKeySemantics checks NaN entries and signed-zero lookup.
func TestMapViewKeySemantics(t *testing.T) {
	nan := math.NaN()
	input := map[float64]int{nan: 1}
	input[nan] = 2
	input[math.Copysign(0, -1)] = 3
	view := Map(ViewOf(FreezeMap(input)))
	if _, found := view.Get(nan); found {
		t.Fatal("NaN unexpectedly compares equal")
	}
	if value, found := view.Get(0); !found || Copy(value) != 3 {
		t.Fatal("signed zero lookup differs")
	}
	if view.Len() != 3 || len(view.Clone()) != 3 {
		t.Fatal("map clone lost NaN entries")
	}
}

// TestViewIteration checks order, repeat use, early stop, and empty views.
func TestViewIteration(t *testing.T) {
	sequence := Slice(ViewOf(FreezeSlice([]int{2, 4, 6})))
	mapping := Map(ViewOf(FreezeMap(map[int]int{1: 2, 3: 4})))
	for range 2 {
		var got []int
		for index, value := range sequence.All() {
			if index != len(got) {
				t.Fatal("slice view iteration order changed")
			}
			got = append(got, Copy(value))
		}
		if !slices.Equal(got, sequence.values) {
			t.Fatal("slice view iteration changed contents")
		}
		entries := make(map[int]int)
		for key, value := range mapping.All() {
			entries[Copy(key)] = Copy(value)
		}
		if !maps.Equal(entries, mapping.values) {
			t.Fatal("map view iteration changed contents")
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
		t.Fatal("view iterator did not stop immediately")
	}
	for range (SliceView[int]{}).All() {
		t.Fatal("empty slice view yielded an entry")
	}
	for range (MapView[int, int]{}).All() {
		t.Fatal("empty map view yielded an entry")
	}
}
