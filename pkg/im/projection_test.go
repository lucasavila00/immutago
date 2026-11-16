package im

import (
	"reflect"
	"strings"
	"testing"
	"unsafe"
)

// TestProjectionUnchecked checks generated-style field reads and shared storage.
func TestProjectionUnchecked(t *testing.T) {
	input := testFrozen{
		Text: strings.Repeat("text", 128), Slice: FreezeSlice([]int{2, 4}),
		Map: FreezeMap(map[int]int{1: 3}), Child: &testChild{Number: 5}, Array: [2]int{6, 7},
	}
	projection := ProjectionOf(&input)
	text := ProjectionOf(projection.Unchecked().Text)
	if unsafe.StringData(text.value) != unsafe.StringData(input.Text) {
		t.Fatal("string read changed storage")
	}
	sequence := ProjectionOf(projection.Unchecked().Slice)
	if &sequence.value.values[0] != &input.Slice.values[0] {
		t.Fatal("slice read changed storage")
	}
	mapping := ProjectionOf(projection.Unchecked().Map)
	if reflect.ValueOf(mapping.value.values).UnsafePointer() != reflect.ValueOf(input.Map.values).UnsafePointer() {
		t.Fatal("map read changed storage")
	}
	pointer := ProjectionOf(projection.Unchecked().Child)
	if pointer.value != input.Child || IsNil(pointer) {
		t.Fatal("pointer read changed identity")
	}
	if projection.Unchecked().Array[1] != 7 {
		t.Fatal("array read changed value")
	}
}

// TestProjectionCopies checks root value copies, scalar copies, and nil tests.
func TestProjectionCopies(t *testing.T) {
	input := [2]int{1, 2}
	projection := ProjectionOf(input)
	input[0] = 9
	copy := Value(projection)
	copy[1] = 8
	if Value(projection) != [2]int{1, 2} || Value(ProjectionOf(3)) != 3 {
		t.Fatal("projection did not keep an independent root value")
	}
	var zero Projection[int]
	var pointer Projection[*testChild]
	if Value(zero) != 0 || !IsNil(pointer) || IsNil(ProjectionOf(&testChild{})) {
		t.Fatal("invalid zero or pointer projection")
	}
}

// TestProjectionPublicAPI checks sealed fields and the fixed method set.
func TestProjectionPublicAPI(t *testing.T) {
	cases := []struct {
		typeOf  reflect.Type
		methods []string
	}{
		{reflect.TypeFor[Projection[int]](), []string{"Unchecked"}},
		{reflect.TypeFor[StringProjection](), []string{"At", "Clone", "Equal", "Is", "Len", "Unchecked"}},
		{reflect.TypeFor[FrozenString](), []string{"At", "Clone", "Equal", "EqualString", "Len", "Projection", "Unchecked"}},
		{reflect.TypeFor[SliceProjection[int]](), []string{"All", "At", "AtPointer", "Clone", "Len"}},
		{reflect.TypeFor[MapProjection[int, int]](), []string{"All", "Clone", "Get", "Len"}},
	}
	for _, test := range cases {
		for i := range test.typeOf.NumField() {
			if test.typeOf.Field(i).IsExported() {
				t.Fatalf("%s exposes a raw field", test.typeOf)
			}
		}
		for _, typ := range []reflect.Type{test.typeOf, reflect.PointerTo(test.typeOf)} {
			if typ.NumMethod() != len(test.methods) {
				t.Fatalf("%s has an unexpected method", typ)
			}
			for i, name := range test.methods {
				if typ.Method(i).Name != name {
					t.Fatalf("%s exposes method %s instead of %s", typ, typ.Method(i).Name, name)
				}
			}
		}
	}
}
