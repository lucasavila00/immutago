package accepted

import (
	"testing"

	"example.com/consumer/api"
	"example.com/consumer/plain"
)

func TestTupleValues(t *testing.T) {
	pool := &api.Pool{}
	first, view, counts, found, last := pool.Read()
	if first != nil || last != nil || !found || view.Text.Unsafe() != "ready" || counts[1].N != 8 {
		t.Fatal("tuple result changed")
	}
	view, found = pool.Named()
	if !found || view.Count.N != 7 {
		t.Fatal("named result changed")
	}
	view, found = Forward()
	if !found || view.Count.N != 7 {
		t.Fatal("forwarded result changed")
	}
	if !BuilderSlots().At(0) {
		t.Fatal("builder slot was not set")
	}
	first, _, last = pool.Errors()
	if first != last || first.(*api.StateError).Values["count"] != 1 {
		t.Fatal("error result changed")
	}
}

func TestLookupAllocations(t *testing.T) {
	prices := plain.Prices{}
	allocations := testing.AllocsPerRun(100, func() {
		price, found := prices.Price("missing")
		if price != 0 || found {
			panic("unexpected price")
		}
	})
	if allocations != 0 {
		t.Fatalf("lookup allocations = %v; want 0", allocations)
	}
}
