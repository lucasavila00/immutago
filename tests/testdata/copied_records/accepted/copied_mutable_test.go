package accepted

import (
	"encoding/json"
	"testing"

	"example.com/consumer/api"
)

func TestDecodeAndMutate(t *testing.T) {
	payload, err := Decode([]byte(`{"ID":"request","Ratio":1.5}`))
	if err != nil || payload.ID != "request" || payload.Ratio.Value() != 1.5 {
		t.Fatalf("JSON decode failed: %v", err)
	}
	copy := payload
	if err := payload.Decode([]byte("2.5")); err != nil {
		t.Fatal(err)
	}
	if copy.Ratio.Value() != 1.5 || payload.Ratio.Value() != 2.5 {
		t.Fatal("decoding changed another copy")
	}
	text, err := DecodeText([]byte("3.5"))
	if err != nil || text.Value() != 3.5 {
		t.Fatalf("text decode failed: %v", err)
	}
	encoded, err := json.Marshal(text)
	if err != nil || string(encoded) != "3.5" {
		t.Fatalf("JSON encode failed: %v", err)
	}
	Nested(&payload)
	if payload.Rows[0].Points[1][0].X != 3 || copy.Rows[0].Points[1][0].X != 0 {
		t.Fatal("nested mutation changed another copy")
	}
}

func TestOrdinaryHolders(t *testing.T) {
	r := request{id: "request"}
	handle(&r, func(id string) (float64, bool) { return 2, id == "request" })
	if r.a != 2 || r.b != 2 {
		t.Fatal("ordinary holder operations failed")
	}
	point := api.Point{X: 1, Y: 2}
	original, changed := Mutate(point)
	if original != point || changed.X != 9 || changed.Y != 4 {
		t.Fatal("pointer mutation changed another copy")
	}
	x, y := Methods(point)
	if x != 4 || y != 1 || Channels(point) != point {
		t.Fatal("ordinary methods or channel operations failed")
	}
	done := make(chan api.Point, 2)
	Concurrent(point, done)
	first, second := <-done, <-done
	if (first != point || second.X != 7) && (second != point || first.X != 7) {
		t.Fatal("concurrent copies changed the source")
	}
}
