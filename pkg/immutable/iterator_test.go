package immutable

import (
	"maps"
	"testing"
)

// TestSliceAll checks ordering, values, and reuse across leaf and tree boundaries.
func TestSliceAll(t *testing.T) {
	for _, size := range []int{0, 1, 32, 33, 1024, 1057, 32801} {
		input := make([]int, size)
		for index := range input {
			input[index] = index * 2
		}
		s := NewSlice(input...)
		iterator := s.All()
		s = s.Append(-1)
		if s.Len() != size+1 {
			t.Fatal("append failed")
		}
		for repeat := 0; repeat < 2; repeat++ {
			count := 0
			for index, value := range iterator {
				if index != count || value != index*2 {
					t.Fatalf("length %d: got index %d value %d at position %d", size, index, value, count)
				}
				count++
			}
			if count != size {
				t.Fatalf("length %d: visited %d values", size, count)
			}
		}
	}
}

// TestSliceAllEarlyStop checks termination in the tree, at block boundaries, and in the tail.
func TestSliceAllEarlyStop(t *testing.T) {
	s := NewSlice(make([]int, 1057)...)
	for _, stop := range []int{1, 31, 32, 33, 1024, 1056, 1057} {
		count := 0
		s.All()(func(int, int) bool {
			count++
			if count > stop {
				t.Fatal("iterator called yield after termination")
			}
			return count < stop
		})
		if count != stop {
			t.Fatalf("stopped after %d values, want %d", count, stop)
		}
	}
}

// TestMapAll checks empty maps, every entry, reuse, and snapshot ownership.
func TestMapAll(t *testing.T) {
	var empty Map[int, int]
	for range empty.All() {
		t.Fatal("empty map yielded an entry")
	}
	input := make(map[int]int, 2000)
	for key := 0; key < 2000; key++ {
		input[key] = key * 2
	}
	m := NewMap(input)
	iterator := m.All()
	m = m.Set(-1, -1)
	if m.Len() != len(input)+1 {
		t.Fatal("insertion failed")
	}
	for repeat := 0; repeat < 2; repeat++ {
		output := make(map[int]int)
		count := 0
		for key, value := range iterator {
			output[key] = value
			count++
		}
		if count != len(input) || !maps.Equal(input, output) {
			t.Fatal("iterator did not yield exactly the original entries")
		}
	}
}

// TestMapAllEarlyStop checks immediate and mid-tree termination, including collision buckets.
func TestMapAllEarlyStop(t *testing.T) {
	m := benchmarkMapOfSize(2000)
	for _, stop := range []int{1, 31, 32, 1000, 2000} {
		count := 0
		m.All()(func(int, int) bool {
			count++
			if count > stop {
				t.Fatal("iterator called yield after termination")
			}
			return count < stop
		})
		if count != stop {
			t.Fatalf("stopped after %d entries, want %d", count, stop)
		}
	}
	collision := mapLeaf[int, int](7, 1, 10)
	collision, _ = setMapNode(collision, 0, 7, 2, 20)
	count := 0
	if yieldMapNode(collision, func(int, int) bool { count++; return false }) || count != 1 {
		t.Fatal("collision iterator did not stop after its first entry")
	}
}

// TestIterationAllocations verifies that direct range loops allocate no heap storage.
func TestIterationAllocations(t *testing.T) {
	s := NewSlice(make([]int, 2000)...)
	m := benchmarkMapOfSize(2000)
	if count := testing.AllocsPerRun(100, func() {
		for index, value := range s.All() {
			benchmarkInt = index + value
		}
	}); count != 0 {
		t.Fatalf("slice iteration allocated %g times", count)
	}
	if count := testing.AllocsPerRun(100, func() {
		for key, value := range m.All() {
			benchmarkInt = key + value
		}
	}); count != 0 {
		t.Fatalf("map iteration allocated %g times", count)
	}
}

// BenchmarkSliceAll measures traversal without copying the backing tree.
func BenchmarkSliceAll(b *testing.B) {
	s := NewSlice(make([]int, 32768)...)
	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		for index, value := range s.All() {
			benchmarkInt = index + value
		}
	}
}

// BenchmarkMapAll measures traversal without allocating an output map.
func BenchmarkMapAll(b *testing.B) {
	m := benchmarkMapOfSize(32768)
	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		for key, value := range m.All() {
			benchmarkInt = key + value
		}
	}
}
