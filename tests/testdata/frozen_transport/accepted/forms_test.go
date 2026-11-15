package accepted

import "testing"

func TestExercise(t *testing.T) {
	if got := Exercise(); got != 59 {
		t.Fatalf("Exercise() = %d; want 59", got)
	}
}
