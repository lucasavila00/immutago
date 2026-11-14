package valid

import (
	"os"
	"testing"
)

func TestMain(m *testing.M) { os.Exit(m.Run()) }
func TestValue(t *testing.T) {
	if (Value{X: 1}).X != 1 {
		t.Fatal("wrong value")
	}
}
