package cell_holder

import (
	"sync"
	"testing"

	api "example.com/consumer/cell_holder_api"
)

func TestSharedSlot(t *testing.T) {
	h := api.New()
	parent, alias := api.Share(h)
	if parent.Child != h || alias != h || api.Identity(h) != h {
		t.Fatal("holder pointer changed")
	}
	var group sync.WaitGroup
	for value := range 4 {
		group.Go(func() {
			parent.Child.Publish(value)
			_ = alias.Read()
		})
	}
	group.Wait()
	h.Publish(7)
	if got := parent.Child.Read(); got != 7 {
		t.Fatalf("stored value = %d; want 7", got)
	}
	var nested api.Nested
	child := api.Child(&nested)
	child.Publish(9)
	if nested.Child.Read() != 9 {
		t.Fatal("nested holder address changed")
	}
}

func TestCapturedHolderVariable(t *testing.T) {
	publish, read := api.CaptureVariable()
	publish(8)
	if got := read(); got != 8 {
		t.Fatalf("captured slot value = %d; want 8", got)
	}
}
