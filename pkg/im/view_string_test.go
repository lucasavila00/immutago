package im

import (
	"strings"
	"testing"
	"unsafe"
)

// TestStringView checks named strings, shared bytes, and independent native copies.
func TestStringView(t *testing.T) {
	type label string
	parent := strings.Repeat("abcdef", 1024)
	input := label(parent[100:107])
	view := Text(ViewOf(input))
	if view.Len() != len(input) || view.At(3) != input[3] ||
		unsafe.StringData(view.value) != unsafe.StringData(string(input)) {
		t.Fatal("text adapter changed storage or bytes")
	}
	first, second := view.Clone(), view.Clone()
	if first != string(input) || second != string(input) {
		t.Fatal("string clone changed bytes")
	}
	if unsafe.StringData(first) == unsafe.StringData(string(input)) ||
		unsafe.StringData(second) == unsafe.StringData(string(input)) ||
		unsafe.StringData(first) == unsafe.StringData(second) {
		t.Fatal("string clone retained shared bytes")
	}
}

// TestStringViewEmpty checks zero values and clones of empty substrings.
func TestStringViewEmpty(t *testing.T) {
	parent := strings.Repeat("x", 1024)
	for _, view := range []StringView{{}, Text(ViewOf("")), Text(ViewOf(parent[512:512]))} {
		clone := view.Clone()
		if view.Len() != 0 || clone != "" {
			t.Fatal("invalid empty string view")
		}
		start := uintptr(unsafe.Pointer(unsafe.StringData(parent)))
		data := uintptr(unsafe.Pointer(unsafe.StringData(clone)))
		if data >= start && data < start+uintptr(len(parent)) {
			t.Fatal("empty clone retained the parent string")
		}
	}
}

// TestStringViewBounds checks negative, end, and empty indices.
func TestStringViewBounds(t *testing.T) {
	for _, view := range []StringView{{}, Text(ViewOf("text"))} {
		for _, index := range []int{-1, view.Len()} {
			func() {
				defer func() {
					if recover() == nil {
						t.Error("out-of-range string view read did not panic")
					}
				}()
				_ = view.At(index)
			}()
		}
	}
}
