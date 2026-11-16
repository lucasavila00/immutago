package accepted

import "testing"

func TestExercise(t *testing.T) {
	if got := Exercise(); got != 59 {
		t.Fatalf("Exercise() = %d; want 59", got)
	}
}

func TestHandleSlots(t *testing.T) {
	if got := HandleSlots(); got != 20 {
		t.Fatalf("HandleSlots() = %d; want 20", got)
	}
}

func TestFreezeSlots(t *testing.T) {
	if got := FreezeSlots(); got != 21 {
		t.Fatalf("FreezeSlots() = %d; want 21", got)
	}
}
