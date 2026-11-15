package supervise

import (
	"encoding/json"
	"testing"
	"testing/synctest"
)

// TestRefIsZero checks unset references and copies through start and stop.
func TestRefIsZero(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		var zero Ref[counter]
		zeroCopy := zero
		if !zero.IsZero() || !zeroCopy.IsZero() {
			t.Fatal("zero Ref or its copy is set")
		}
		reserved := Reserve[counter]()
		reservedCopy := reserved
		if reserved.IsZero() || reservedCopy.IsZero() {
			t.Fatal("reserved Ref or its copy is zero before start")
		}
		expectPanic(t, "supervise: Ref not started", func() { reserved.Borrow() })
		root := NewRoot(nil)
		if err := StartReserved(root, reserved, NoArgs{}); err != nil {
			t.Fatal(err)
		}
		if reserved.IsZero() || reservedCopy.IsZero() {
			t.Fatal("reserved Ref or its copy is zero after start")
		}
		started, err := Start[counter](root, NoArgs{})
		if err != nil {
			t.Fatal(err)
		}
		startedCopy := started
		if started.IsZero() || startedCopy.IsZero() {
			t.Fatal("started Ref or its copy is zero")
		}
		<-root.Stop()
		if started.IsZero() || startedCopy.IsZero() || reserved.IsZero() || reservedCopy.IsZero() {
			t.Fatal("stopped Ref or its copy is zero")
		}
	})
}

// TestRefIsZeroAllocations checks both results without service access.
func TestRefIsZeroAllocations(t *testing.T) {
	for _, ref := range []Ref[counter]{{}, Reserve[counter]()} {
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
	for _, test := range []struct {
		ref  Ref[counter]
		want string
	}{
		{Ref[counter]{}, `{}`},
		{Reserve[counter](), `{"cache":{}}`},
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
