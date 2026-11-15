package im

import (
	"testing"
	"unsafe"
)

type testCopiedChild struct {
	Copied
	Count uint64
}

type testCopiedRecord struct {
	Copied
	Values   [16]uint64
	Children [2]testCopiedChild
}

var copiedRecordSink testCopiedRecord

// TestCopiedRepresentation checks the marker's size and an ordinary record layout.
func TestCopiedRepresentation(t *testing.T) {
	type plain struct {
		Values   [16]uint64
		Children [2]uint64
	}
	if unsafe.Sizeof(Copied{}) != 0 || unsafe.Sizeof(testCopiedRecord{}) != unsafe.Sizeof(plain{}) {
		t.Fatal("Copied marker changed the data size")
	}
}

// TestCopiedAllocations checks complete record reads without a new allocation.
func TestCopiedAllocations(t *testing.T) {
	value := ProjectionOf(testCopiedRecord{Values: [16]uint64{3}})
	if got := testing.AllocsPerRun(100, func() { copiedRecordSink = Copy(value) }); got != 0 {
		t.Fatalf("Copy allocations = %g; want zero", got)
	}
}

// TestCopiedClones checks independent slice and map slots containing complete records.
func TestCopiedClones(t *testing.T) {
	value := testCopiedRecord{Values: [16]uint64{3}, Children: [2]testCopiedChild{{Count: 4}}}
	sequence := FreezeSlice([]testCopiedRecord{value}).Projection()
	values := sequence.Clone()
	values[0] = testCopiedRecord{}
	if Copy(sequence.At(0)) != value {
		t.Fatal("slice clone changed a stored record")
	}
	mapping := FreezeMap(map[testCopiedChild]testCopiedRecord{{Count: 1}: value}).Projection()
	entries := mapping.Clone()
	entries[testCopiedChild{Count: 1}] = testCopiedRecord{}
	read, found := mapping.Get(testCopiedChild{Count: 1})
	if !found || Copy(read) != value {
		t.Fatal("map clone changed a stored record")
	}
}

// BenchmarkCopiedRecord compares native assignment with a projection value copy.
func BenchmarkCopiedRecord(b *testing.B) {
	value := testCopiedRecord{Values: [16]uint64{3}}
	projection := ProjectionOf(value)
	b.Run("native", func(b *testing.B) {
		b.ReportAllocs()
		for b.Loop() {
			copiedRecordSink = value
		}
	})
	b.Run("projection", func(b *testing.B) {
		b.ReportAllocs()
		for b.Loop() {
			copiedRecordSink = Copy(projection)
		}
	})
}
