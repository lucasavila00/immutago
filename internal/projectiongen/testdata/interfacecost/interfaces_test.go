package interfacecost

import (
	"testing"
	"unsafe"
)

func TestProjectionCost(t *testing.T) {
	if unsafe.Sizeof(ScorerProjection{}) != 2*unsafe.Sizeof(uintptr(0)) {
		t.Fatal("interface projection has extra storage")
	}
	current.Store(Model{Value: &Number{N: 7}})
	if count := testing.AllocsPerRun(100, read); count != 0 {
		t.Fatalf("projected interface read allocations = %g", count)
	}
	if result != 7 {
		t.Fatal(result)
	}
}

func BenchmarkDirectInterface(b *testing.B) {
	current.Store(Model{Value: &Number{N: 7}})
	value := current.LoadShared().Value
	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		result = value.Score()
	}
}

func BenchmarkProjectedInterface(b *testing.B) {
	current.Store(Model{Value: &Number{N: 7}})
	value := current.Load().Value()
	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		result = value.Score()
	}
}
