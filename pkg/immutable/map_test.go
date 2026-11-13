package immutable

import (
	"maps"
	"math"
	"math/rand/v2"
	"sync"
	"testing"
)

// TestMapZeroAndCopies checks empty maps, input/output copies, and independent update branches.
func TestMapZeroAndCopies(t *testing.T) {
	var empty Map[string, int]
	if value, ok := empty.Get("missing"); value != 0 || ok || empty.Len() != 0 {
		t.Fatal("invalid zero map")
	}
	if empty.Delete("missing").Len() != 0 || len(empty.Values()) != 0 {
		t.Fatal("invalid empty operation")
	}
	input := map[string]int{"one": 1, "two": 2}
	original := NewMap(input)
	input["one"] = 99
	output := original.Values()
	output["two"] = 99
	left := original.Set("one", 10)
	right := original.Set("three", 3)
	deleted := original.Delete("one")
	if !maps.Equal(original.Values(), map[string]int{"one": 1, "two": 2}) {
		t.Fatal("a write changed the original")
	}
	if value, ok := left.Get("one"); !ok || value != 10 || left.Len() != 2 {
		t.Fatal("replacement failed")
	}
	if value, ok := right.Get("three"); !ok || value != 3 || right.Len() != 3 {
		t.Fatal("insertion failed")
	}
	if _, ok := deleted.Get("one"); ok || deleted.Len() != 1 {
		t.Fatal("deletion failed")
	}
	if original.Delete("missing").root != original.root {
		t.Fatal("missing deletion copied the tree")
	}
	if NewMap[string, int](nil).Len() != 0 {
		t.Fatal("nil input was not empty")
	}
}

// TestMapRandomVersions compares random updates and retained snapshots with native maps.
func TestMapRandomVersions(t *testing.T) {
	random := rand.New(rand.NewPCG(56, 78))
	versions := []Map[int, int]{{}}
	models := []map[int]int{{}}
	for step := 0; step < 3000; step++ {
		parent := random.IntN(len(versions))
		current, model := versions[parent], maps.Clone(models[parent])
		key, value := random.IntN(100), random.IntN(10000)
		if random.IntN(3) == 0 {
			current = current.Delete(key)
			delete(model, key)
		} else {
			current = current.Set(key, value)
			model[key] = value
		}
		versions = append(versions, current)
		models = append(models, model)
	}
	for index, version := range versions {
		if version.Len() != len(models[index]) || !maps.Equal(version.Values(), models[index]) {
			t.Fatalf("version %d differs from model", index)
		}
		for key, want := range models[index] {
			if got, ok := version.Get(key); !ok || got != want {
				t.Fatalf("version %d key %d: got %d, %t; want %d", index, key, got, ok, want)
			}
		}
	}
}

// TestMapLargeDeletion checks growth, shrinkage, and reinsertion after the last key is removed.
func TestMapLargeDeletion(t *testing.T) {
	var current Map[int, int]
	for key := 0; key < 10000; key++ {
		current = current.Set(key, key*2)
	}
	original := current
	for key := 0; key < 10000; key++ {
		if got, ok := current.Get(key); !ok || got != key*2 {
			t.Fatalf("missing key %d before deletion", key)
		}
		current = current.Delete(key)
	}
	if current.Len() != 0 || current.root != nil || original.Len() != 10000 {
		t.Fatal("deletion corrupted a version")
	}
	if got, ok := current.Set(1, 2).Get(1); !ok || got != 2 {
		t.Fatal("reinsertion into cleared map failed")
	}
}

// TestMapFloatKeys verifies NaN and signed-zero behavior against Go map equality.
func TestMapFloatKeys(t *testing.T) {
	nan := math.NaN()
	current := NewMap(map[float64]int{nan: 1, 0: 2})
	current = current.Set(nan, 3)
	if current.Len() != 3 {
		t.Fatal("NaN insertion replaced an unequal key")
	}
	if _, ok := current.Get(nan); ok || current.Delete(nan).Len() != 3 {
		t.Fatal("NaN compared equal")
	}
	current = current.Set(math.Copysign(0, -1), 4)
	if got, ok := current.Get(0); !ok || got != 4 || current.Len() != 3 {
		t.Fatal("signed zeros did not compare equal")
	}
	if len(current.Values()) != 3 {
		t.Fatal("Values dropped NaN entries")
	}
}

// TestMapReadAllocations verifies that successful and missing lookups allocate no memory.
func TestMapReadAllocations(t *testing.T) {
	current := NewMap(map[string]int{"value": 42})
	if allocations := testing.AllocsPerRun(100, func() {
		_, _ = current.Get("value")
		_, _ = current.Get("missing")
	}); allocations != 0 {
		t.Fatalf("Get allocated %g times", allocations)
	}
}

// TestCollectionConcurrentVersions checks shared reads and independent updates under the race detector.
func TestCollectionConcurrentVersions(t *testing.T) {
	s := NewSlice(make([]int, 2000)...)
	m := NewMap(map[int]int{1: 2})
	var group sync.WaitGroup
	for worker := 0; worker < 8; worker++ {
		group.Go(func() {
			for index := 0; index < 100; index++ {
				_ = s.Set(index, 1).Append(index).At(index)
				_, _ = m.Set(index, 2).Delete(1).Get(index)
			}
		})
	}
	group.Wait()
	if s.At(0) != 0 || m.Len() != 1 {
		t.Fatal("concurrent updates changed shared values")
	}
}
