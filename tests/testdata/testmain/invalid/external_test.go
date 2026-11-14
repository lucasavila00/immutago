package invalid_test

import (
	"example.com/consumer/invalid"
	"testing"
)

func TestExternal(t *testing.T) { p := &invalid.Value{}; p.X = 3 }
