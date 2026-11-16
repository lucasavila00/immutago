package supervise

import (
	"encoding/json"
	"testing"
	"testing/synctest"
	"unsafe"
)

// TestRefIsZero checks unset references and copies through start and stop.
func TestRefIsZero(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		var zero Ref[counter]
		zeroCopy := zero
		if !zero.IsZero() || !zeroCopy.IsZero() {
			t.Fatal("zero Ref or its copy is set")
		}
		root := NewRoot(nil)
		started, err := Start[counter](root, NoArgs{})
		if err != nil {
			t.Fatal(err)
		}
		startedCopy := started
		if started.IsZero() || startedCopy.IsZero() {
			t.Fatal("started Ref or its copy is zero")
		}
		<-root.Stop()
		if started.IsZero() || startedCopy.IsZero() {
			t.Fatal("stopped Ref or its copy is zero")
		}
	})
}

// TestZeroRefBorrowPanics checks that an unset reference has no service to select.
func TestZeroRefBorrowPanics(t *testing.T) {
	var ref Ref[counter]
	expectPanic(t, "supervise: zero Ref", func() { ref.Borrow() })
}

// TestRefIsZeroAllocations checks both results without service access.
func TestRefIsZeroAllocations(t *testing.T) {
	root := NewRoot(nil)
	defer func() { <-root.Stop() }()
	started, err := Start[counter](root, NoArgs{})
	if err != nil {
		t.Fatal(err)
	}
	for _, ref := range []Ref[counter]{{}, started} {
		var got bool
		if allocations := testing.AllocsPerRun(100, func() { got = ref.IsZero() }); allocations != 0 {
			t.Fatalf("IsZero allocations = %v, want 0", allocations)
		}
		if got != (ref == (Ref[counter]{})) {
			t.Fatal("IsZero differs from comparison with the zero Ref")
		}
	}
}

// TestRefIsZeroOmitZero checks that JSON omits only an unset reference field.
func TestRefIsZeroOmitZero(t *testing.T) {
	root := NewRoot(nil)
	defer func() { <-root.Stop() }()
	started, err := Start[counter](root, NoArgs{})
	if err != nil {
		t.Fatal(err)
	}
	for _, test := range []struct {
		ref  Ref[counter]
		want string
	}{
		{Ref[counter]{}, `{}`},
		{started, `{"cache":{}}`},
	} {
		value := struct {
			Cache Ref[counter] `json:"cache,omitzero"`
		}{Cache: test.ref}
		encoded, err := json.Marshal(value)
		if err != nil {
			t.Fatal(err)
		}
		if string(encoded) != test.want {
			t.Fatalf("JSON = %s, want %s", encoded, test.want)
		}
	}
}

// TestRefRepresentation checks that the marker and payload keep a single-pointer handle.
func TestRefRepresentation(t *testing.T) {
	if unsafe.Sizeof(Ref[counter]{}) != unsafe.Sizeof((*counter)(nil)) {
		t.Fatal("Ref is larger than one pointer")
	}
	root := NewRoot(nil)
	defer func() { <-root.Stop() }()
	ref, err := Start[counter](root, NoArgs{})
	if err != nil {
		t.Fatal(err)
	}
	original := ref.Borrow()
	var got *counter
	if allocations := testing.AllocsPerRun(100, func() { got = ref.Borrow() }); allocations != 0 || got != original {
		t.Fatalf("Borrow allocations=%v, stable=%v", allocations, got == original)
	}
	<-root.Stop()
	if ref.Borrow() != original {
		t.Fatal("Stop changed the pointer")
	}
}
