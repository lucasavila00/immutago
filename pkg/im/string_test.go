package im

import (
	"strings"
	"testing"
	"unsafe"
)

// TestFreezeString checks shared bytes, equality, and independent native copies.
func TestFreezeString(t *testing.T) {
	parent := strings.Repeat("abcdef", 1024)
	input := parent[100:107]
	frozen := FreezeString(input)
	view := frozen.ByValue()
	if frozen.Len() != len(input) || frozen.At(3) != input[3] || frozen != FreezeString(input) {
		t.Fatal("frozen bytes changed")
	}
	for _, text := range []string{frozen.value, view.value, FreezeByValueString(input).value} {
		if unsafe.StringData(text) != unsafe.StringData(input) {
			t.Fatal("freeze or view creation copied bytes")
		}
	}
	first, second := frozen.Clone(), view.Clone()
	if first != input || second != input {
		t.Fatal("clone bytes changed")
	}
	if unsafe.StringData(first) == unsafe.StringData(input) || unsafe.StringData(second) == unsafe.StringData(input) ||
		unsafe.StringData(first) == unsafe.StringData(second) {
		t.Fatal("clone retained shared storage")
	}
}

// TestFrozenStringBounds checks both string families at negative, end, and empty indices.
func TestFrozenStringBounds(t *testing.T) {
	frozen := FreezeString("text")
	for _, read := range []func(int) byte{frozen.At, frozen.ByValue().At} {
		for _, index := range []int{-1, frozen.Len()} {
			func() {
				defer func() {
					if recover() == nil {
						t.Error("out-of-range access did not panic")
					}
				}()
				_ = read(index)
			}()
		}
	}
	for _, read := range []func(int) byte{(FrozenString{}).At, (FrozenByValueString{}).At} {
		func() {
			defer func() {
				if recover() == nil {
					t.Error("empty indexing did not panic")
				}
			}()
			_ = read(0)
		}()
	}
}

// TestFrozenStringEmpty checks empty input, zero values, and empty native copies.
func TestFrozenStringEmpty(t *testing.T) {
	parent := strings.Repeat("x", 1024)
	var zero FrozenString
	for _, frozen := range []FrozenString{zero, FreezeString(""), FreezeString(parent[512:512])} {
		if frozen.Len() != 0 || frozen.Clone() != "" || frozen.ByValue().Clone() != "" {
			t.Fatal("invalid empty string")
		}
	}
	var view FrozenByValueString
	if view.Len() != 0 || view.Clone() != "" || FreezeByValueString("").Clone() != "" {
		t.Fatal("invalid empty string view")
	}
}

var stringSink string
var frozenStringSink FrozenString
var byValueStringSink FrozenByValueString
var stringByteSink byte
var stringLenSink int

// TestFrozenStringAllocations checks zero-copy construction and explicit copies that escape.
func TestFrozenStringAllocations(t *testing.T) {
	input := strings.Repeat("text", 32)
	frozen := FreezeString(input)
	view := frozen.ByValue()
	cases := []struct {
		name string
		want float64
		run  func()
	}{
		{"freeze", 0, func() { frozenStringSink = FreezeString(input) }},
		{"freeze by value", 0, func() { byValueStringSink = FreezeByValueString(input) }},
		{"view", 0, func() { byValueStringSink = frozen.ByValue() }},
		{"clone", 1, func() { stringSink = frozen.Clone() }},
		{"view clone", 1, func() { stringSink = view.Clone() }},
		{"read", 0, func() { stringByteSink = frozen.At(3); stringLenSink = frozen.Len() }},
		{"view read", 0, func() { stringByteSink = view.At(3); stringLenSink = view.Len() }},
		{"empty clone", 0, func() { stringSink = (FrozenString{}).Clone() }},
		{"empty view clone", 0, func() { stringSink = (FrozenByValueString{}).Clone() }},
	}
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			if got := testing.AllocsPerRun(100, test.run); got != test.want {
				t.Fatalf("allocations = %g; want %g", got, test.want)
			}
		})
	}
}

// BenchmarkFreezeString measures wrapping an existing native string without a copy.
func BenchmarkFreezeString(b *testing.B) {
	input := strings.Repeat("x", 1024)
	b.ReportAllocs()
	for b.Loop() {
		frozenStringSink = FreezeString(input)
	}
}

// BenchmarkStringClone measures native string copy-out.
func BenchmarkStringClone(b *testing.B) {
	frozen := FreezeString(strings.Repeat("x", 1024))
	b.ReportAllocs()
	for b.Loop() {
		stringSink = frozen.Clone()
	}
}

// BenchmarkStringRead measures byte and length reads without a copy.
func BenchmarkStringRead(b *testing.B) {
	frozen := FreezeString(strings.Repeat("x", 1024))
	b.ReportAllocs()
	for b.Loop() {
		stringByteSink = frozen.At(512)
		stringLenSink = frozen.Len()
	}
}

// BenchmarkStringByValue measures creation of a local view without a copy.
func BenchmarkStringByValue(b *testing.B) {
	frozen := FreezeString(strings.Repeat("x", 1024))
	b.ReportAllocs()
	for b.Loop() {
		byValueStringSink = frozen.ByValue()
	}
}

// TestEmptyStringClone drops any source pointer in an empty substring.
func TestEmptyStringClone(t *testing.T) {
	parent := strings.Repeat("x", 1024)
	empty := parent[512:512]
	for _, clone := range []string{FreezeString(empty).Clone(), FreezeByValueString(empty).Clone()} {
		if clone != "" {
			t.Fatal("empty substring clone changed the value")
		}
		if source := unsafe.StringData(empty); source != nil && unsafe.StringData(clone) == source {
			t.Fatal("empty substring clone retained the source pointer")
		}
	}
}
