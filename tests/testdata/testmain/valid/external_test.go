package valid_test

import (
	"example.com/consumer/valid"
	"testing"
)

func TestExternal(t *testing.T) {
	if (valid.Value{X: 2}).X != 2 {
		t.Fatal("wrong value")
	}
}
