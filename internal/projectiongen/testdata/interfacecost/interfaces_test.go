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
	if count := testing.AllocsPerRun(100, readResults); count != 0 {
		t.Fatalf("projected interface tuple allocations = %g", count)
	}
	if result != 7 || !found || len(output) != 1 || &output[0] != &buffer[0] || failure != nil {
		t.Fatal("tuple projection changed the result or caller buffer")
	}
}

func BenchmarkDirectInterface(b *testing.B) {
	current.Store(Model{Value: &Number{N: 7}})
	value := current.Load().Unchecked().Value
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
