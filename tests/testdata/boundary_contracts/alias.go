package contract

import (
	"example.com/consumer/model"
	"unsafe"
)

var _ model.Scalar[int]
var _ unsafe.Pointer
