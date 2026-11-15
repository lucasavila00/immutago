package im

import (
	"strings"
	"testing"
	"unsafe"
)

// TestFreezeString checks bytes, equality, and detachment from a larger allocation.
func TestFreezeString(t *testing.T) {
	parent := strings.Repeat("abcdef", 1024)
	input := parent[100:107]
	frozen := FreezeString(input)
	if frozen.Len() != len(input) || frozen.At(3) != input[3] || frozen != FreezeString(input) {
		t.Fatal("frozen bytes changed")
	}
	if unsafe.StringData(frozen.value) == unsafe.StringData(input) {
		t.Fatal("freeze retained input storage")
	}
	first, second := frozen.Clone(), frozen.Clone()
	if first != input || second != input {
		t.Fatal("clone bytes changed")
	}
	if unsafe.StringData(first) == unsafe.StringData(frozen.value) ||
		unsafe.StringData(first) == unsafe.StringData(second) {
		t.Fatal("clone retained shared storage")
	}
	for _, index := range []int{-1, len(input)} {
		func() {
			defer func() {
				if recover() == nil {
					t.Error("out-of-range access did not panic")
				}
			}()
			_ = frozen.At(index)
		}()
	}
}

// TestFrozenStringEmpty checks that empty strings do not retain their input allocation.
func TestFrozenStringEmpty(t *testing.T) {
	parent := strings.Repeat("x", 1024)
	var zero FrozenString
	for _, frozen := range []FrozenString{zero, FreezeString(""), FreezeString(parent[512:512])} {
		if frozen.Len() != 0 || frozen.Clone() != "" || unsafe.StringData(frozen.value) != nil {
			t.Fatal("empty frozen string retains storage")
		}
	}
	defer func() {
		if recover() == nil {
			t.Fatal("empty indexing did not panic")
		}
	}()
	_ = zero.At(0)
}

var stringSink string
var frozenStringSink FrozenString
var stringByteSink byte
var stringLenSink int

// TestFrozenStringAllocations checks copies that escape and allocation-free reads.
func TestFrozenStringAllocations(t *testing.T) {
	input := strings.Repeat("text", 32)
	frozen := FreezeString(input)
	cases := []struct {
		name string
		want float64
		run  func()
	}{
		{"freeze", 1, func() { frozenStringSink = FreezeString(input) }},
		{"clone", 1, func() { stringSink = frozen.Clone() }},
		{"read", 0, func() { stringByteSink = frozen.At(3); stringLenSink = frozen.Len() }},
		{"empty freeze", 0, func() { frozenStringSink = FreezeString("") }},
		{"empty clone", 0, func() { stringSink = (FrozenString{}).Clone() }},
	}
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			if got := testing.AllocsPerRun(100, test.run); got != test.want {
				t.Fatalf("allocations = %g; want %g", got, test.want)
			}
		})
	}
}

// BenchmarkFreezeString measures the copy that detaches a substring from its parent.
func BenchmarkFreezeString(b *testing.B) {
	parent := strings.Repeat("x", 1<<20)
	input := parent[512:1536]
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
