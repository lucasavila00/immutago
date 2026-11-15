package im

import (
	"strings"
	"testing"
	"unsafe"
)

// TestStringProjection checks named strings, shared bytes, and independent native copies.
func TestStringProjection(t *testing.T) {
	type label string
	parent := strings.Repeat("abcdef", 1024)
	input := label(parent[100:107])
	projection := StringOf(ProjectionOf(input))
	if projection.Len() != len(input) || projection.At(3) != input[3] ||
		unsafe.StringData(projection.value) != unsafe.StringData(string(input)) {
		t.Fatal("text adapter changed storage or bytes")
	}
	first, second := projection.Clone(), projection.Clone()
	if first != string(input) || second != string(input) {
		t.Fatal("string clone changed bytes")
	}
	if unsafe.StringData(first) == unsafe.StringData(string(input)) ||
		unsafe.StringData(second) == unsafe.StringData(string(input)) ||
		unsafe.StringData(first) == unsafe.StringData(second) {
		t.Fatal("string clone retained shared bytes")
	}
}

// TestStringProjectionEmpty checks zero values and clones of empty substrings.
func TestStringProjectionEmpty(t *testing.T) {
	parent := strings.Repeat("x", 1024)
	for _, projection := range []StringProjection{{}, ProjectString(""), ProjectString(parent[512:512])} {
		clone := projection.Clone()
		if projection.Len() != 0 || clone != "" {
			t.Fatal("invalid empty string projection")
		}
		start := uintptr(unsafe.Pointer(unsafe.StringData(parent)))
		data := uintptr(unsafe.Pointer(unsafe.StringData(clone)))
		if data >= start && data < start+uintptr(len(parent)) {
			t.Fatal("empty clone retained the parent string")
		}
	}
}

// TestStringProjectionBounds checks negative, end, and empty indices.
func TestStringProjectionBounds(t *testing.T) {
	for _, projection := range []StringProjection{{}, StringOf(ProjectionOf("text"))} {
		for _, index := range []int{-1, projection.Len()} {
			func() {
				defer func() {
					if recover() == nil {
						t.Error("out-of-range string projection read did not panic")
					}
				}()
				_ = projection.At(index)
			}()
		}
	}
}
