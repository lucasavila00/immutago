package im

import (
	"strings"
	"testing"
	"unsafe"
)

var frozenStringSink FrozenString

// TestFrozenString checks shared construction and detached copies from both APIs.
func TestFrozenString(t *testing.T) {
	parent := strings.Repeat("abcdef", 1024)
	input := parent[100:107]
	frozen := FreezeString(input)
	view := String(ViewOf(frozen))
	if unsafe.StringData(frozen.value) != unsafe.StringData(input) ||
		unsafe.StringData(view.value) != unsafe.StringData(input) {
		t.Fatal("string construction copied backing bytes")
	}
	if frozen.Len() != len(input) || view.Len() != len(input) {
		t.Fatal("string length changed")
	}
	for index := range len(input) {
		if frozen.At(index) != input[index] || view.At(index) != input[index] {
			t.Fatal("string byte changed")
		}
	}
	for _, clone := range []string{frozen.Clone(), view.Clone()} {
		if clone != input || unsafe.StringData(clone) == unsafe.StringData(input) {
			t.Fatal("clone must preserve bytes in independent storage")
		}
	}
}

// TestFrozenStringEmptyAndBounds checks the zero value and native index panics.
func TestFrozenStringEmptyAndBounds(t *testing.T) {
	for _, value := range []FrozenString{{}, FreezeString(""), FreezeString("abc")} {
		if value.Len() == 0 && (value.Clone() != "" || String(ViewOf(value)).Len() != 0) {
			t.Fatal("invalid empty string")
		}
		for _, index := range []int{-1, value.Len()} {
			func() {
				defer func() {
					if recover() == nil {
						t.Error("out-of-range read did not panic")
					}
				}()
				_ = value.At(index)
			}()
		}
	}
}

// TestFrozenStringAllocations separates shared reads from explicit copy-out.
func TestFrozenStringAllocations(t *testing.T) {
	input := strings.Repeat("text", 128)
	frozen := FreezeString(input)
	cases := []struct {
		name string
		run  func()
	}{
		{"freeze", func() { frozenStringSink = FreezeString(input) }},
		{"view", func() { stringViewSink = String(ViewOf(frozen)) }},
		{"read", func() { stringByteSink = frozen.At(1); benchmarkInt = frozen.Len() }},
	}
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			if got := testing.AllocsPerRun(100, test.run); got != 0 {
				t.Fatalf("allocations = %g; want zero", got)
			}
		})
	}
	if got := testing.AllocsPerRun(100, func() { stringCloneSink = frozen.Clone() }); got != 1 {
		t.Fatalf("clone allocations = %g; want one", got)
	}
	if got := testing.AllocsPerRun(100, func() { stringCloneSink = (FrozenString{}).Clone() }); got != 0 {
		t.Fatalf("empty clone allocations = %g; want zero", got)
	}
}

// TestStringComparisonAndUnsafe checks native comparisons and explicit shared access.
func TestStringComparisonAndUnsafe(t *testing.T) {
	input := strings.Repeat("abc", 128)
	frozen := FreezeString(input)
	view := String(ViewOf(frozen))
	for _, other := range []string{"", input, "other", input[:len(input)-1]} {
		want := input == other
		if frozen.Equal(FreezeString(other)) != want || frozen.EqualString(other) != want ||
			view.Equal(Text(ViewOf(other))) != want || view.EqualString(other) != want {
			t.Fatal("string comparison differs from native equality")
		}
	}
	if unsafe.StringData(frozen.Unsafe()) != unsafe.StringData(input) ||
		unsafe.StringData(view.Unsafe()) != unsafe.StringData(input) {
		t.Fatal("Unsafe copied backing storage")
	}
	if got := testing.AllocsPerRun(100, func() {
		benchmarkFound = view.EqualString(input) && view.Equal(String(ViewOf(frozen)))
		benchmarkFound = benchmarkFound && frozen.EqualString(input) && frozen.Equal(frozen)
		stringCloneSink = view.Unsafe()
		stringCloneSink = frozen.Unsafe()
	}); got != 0 {
		t.Fatalf("comparison and Unsafe allocations = %g; want zero", got)
	}
}

// BenchmarkStringComparison compares native string equality with the view method.
func BenchmarkStringComparison(b *testing.B) {
	left := strings.Repeat("abc", 128)
	right := strings.Clone(left)
	view := Text(ViewOf(left))
	b.Run("native", func(b *testing.B) {
		b.ReportAllocs()
		for b.Loop() {
			benchmarkFound = left == right
		}
	})
	b.Run("view", func(b *testing.B) {
		b.ReportAllocs()
		for b.Loop() {
			benchmarkFound = view.EqualString(right)
		}
	})
}
