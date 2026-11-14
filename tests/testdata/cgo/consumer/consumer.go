package consumer

import (
	"example.com/consumer/cgolib"
	"github.com/lucasavila00/immutago/pkg/im"
)

type Value struct {
	im.Shared
	N int
}

// Read must retain the dependency's cgo rejection through either driver.
func Read() Value { return Value{N: cgolib.Add(1, 2)} }
