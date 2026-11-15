package im

import (
	"reflect"
	"strings"
	"testing"
	"unsafe"
)

// TestViewSelect checks one synchronous projection call and shared field storage.
func TestViewSelect(t *testing.T) {
	input := testFrozen{
		Text: strings.Repeat("text", 128), Slice: FreezeSlice([]int{2, 4}),
		Map: FreezeMap(map[int]int{1: 3}), Child: &testChild{Number: 5}, Array: [2]int{6, 7},
	}
	view := ViewOf(input)
	calls := 0
	text := Select(view, func(v testFrozen) string { calls++; return v.Text })
	if calls != 1 || unsafe.StringData(text.value) != unsafe.StringData(input.Text) {
		t.Fatal("string projection changed storage or call count")
	}
	sequence := Select(view, func(v testFrozen) FrozenSlice[int] { calls++; return v.Slice })
	if calls != 2 || &sequence.value.values[0] != &input.Slice.values[0] {
		t.Fatal("slice projection changed storage or call count")
	}
	mapping := Select(view, func(v testFrozen) FrozenMap[int, int] { calls++; return v.Map })
	if calls != 3 || reflect.ValueOf(mapping.value.values).UnsafePointer() !=
		reflect.ValueOf(input.Map.values).UnsafePointer() {
		t.Fatal("map projection changed storage or call count")
	}
	pointer := Select(view, func(v testFrozen) *testChild { calls++; return v.Child })
	if calls != 4 || pointer.value != input.Child || IsNil(pointer) {
		t.Fatal("pointer projection changed identity or call count")
	}
	index := 1
	number := Select(view, func(v testFrozen) int { calls++; return v.Array[index] })
	if calls != 5 || Copy(number) != 7 {
		t.Fatal("array projection changed value or call count")
	}
}

// TestViewCopies checks root value copies, scalar copies, and nil tests.
func TestViewCopies(t *testing.T) {
	input := [2]int{1, 2}
	view := ViewOf(input)
	input[0] = 9
	copy := Copy(view)
	copy[1] = 8
	if Copy(view) != [2]int{1, 2} || Copy(ViewOf(3)) != 3 {
		t.Fatal("view did not keep an independent root value")
	}
	var zero View[int]
	var pointer View[*testChild]
	if Copy(zero) != 0 || !IsNil(pointer) || IsNil(ViewOf(&testChild{})) {
		t.Fatal("invalid zero or pointer view")
	}
}

// TestViewPublicAPI prevents raw fields and unwrapping methods on read views.
func TestViewPublicAPI(t *testing.T) {
	cases := []struct {
		typeOf  reflect.Type
		methods []string
	}{
		{reflect.TypeFor[View[int]](), nil},
		{reflect.TypeFor[StringView](), []string{"At", "Clone", "Equal", "EqualString", "Len", "Unsafe"}},
		{reflect.TypeFor[SliceView[int]](), []string{"All", "At", "AtPointer", "Clone", "Len"}},
		{reflect.TypeFor[MapView[int, int]](), []string{"All", "Clone", "Get", "Len"}},
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
