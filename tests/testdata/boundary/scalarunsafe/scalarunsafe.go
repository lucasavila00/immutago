package scalarunsafe

import (
	"example.com/consumer/hidden"
	"unsafe"
)

var _ unsafe.Pointer

func Number() int { return hidden.Number() }
