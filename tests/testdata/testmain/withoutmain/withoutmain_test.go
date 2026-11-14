package withoutmain

import "testing"

func TestValue(t *testing.T) {
	if (Value{X: 1}).X != 1 {
		t.Fatal("wrong value")
	}
}
