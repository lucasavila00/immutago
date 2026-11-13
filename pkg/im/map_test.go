package im

import (
	"maps"
	"math"
	"testing"
)

// TestFreezeMap checks native lookup and independent writable clones.
func TestFreezeMap(t *testing.T) {
	input := map[string]int{"one": 1, "two": 2}
	frozen := FreezeMap(input)
	if value, found := frozen.Get("two"); !found || value != 2 || frozen.Len() != 2 {
		t.Fatal("map contents changed")
	}
	if value, found := frozen.Get("missing"); found || value != 0 {
		t.Fatal("missing key did not return zero,false")
	}
	first, second := frozen.Clone(), frozen.Clone()
	if !maps.Equal(first, input) {
		t.Fatal("clone contents changed")
	}
	first["one"] = 9
	delete(first, "two")
	first["three"] = 3
	if !maps.Equal(second, input) || frozen.Len() != 2 {
		t.Fatal("clone shares mutable storage")
	}
	if value, _ := frozen.Get("one"); value != 1 {
		t.Fatal("clone changed frozen entry")
	}
}

// TestMapZeroValues checks nil preservation and usable empty non-nil clones.
func TestMapZeroValues(t *testing.T) {
	var zero FrozenMap[int, int]
	if zero.Len() != 0 || zero.Clone() != nil || FreezeMap[int, int](nil).Clone() != nil {
		t.Fatal("invalid zero map")
	}
	if value, found := zero.Get(1); found || value != 0 {
		t.Fatal("nil map lookup changed")
	}
	empty := FreezeMap(map[int]int{}).Clone()
	if empty == nil {
		t.Fatal("empty clone lost nil distinction")
	}
	empty[1] = 2
}

// TestMapNativeKeySemantics checks NaN entries, signed zero, and composite comparable keys.
func TestMapNativeKeySemantics(t *testing.T) {
	nan := math.NaN()
	input := map[float64]int{nan: 1}
	input[nan] = 2
	input[math.Copysign(0, -1)] = 3
	frozen := FreezeMap(input)
	if _, found := frozen.Get(nan); found {
		t.Fatal("NaN unexpectedly compares equal")
	}
	if value, found := frozen.Get(0); !found || value != 3 {
		t.Fatal("signed zero lookup differs")
	}
	if frozen.Len() != 3 || len(frozen.Clone()) != 3 {
		t.Fatal("NaN entries were lost")
	}
	type key struct {
		N int
		S string
	}
	composite := FreezeMap(map[key]int{{1, "a"}: 7})
	if value, found := composite.Get(key{1, "a"}); !found || value != 7 {
		t.Fatal("composite key lookup differs")
	}
}
