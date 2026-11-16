package local_pointer_field

//go:generate go run github.com/lucasavila00/immutago/cmd/immutago-gen

import (
	"github.com/lucasavila00/immutago/pkg/im"
)

type Local struct {
	im.Local
	Value int
}
type Stored struct {
	im.Frozen
	Pointer *Local
}

func stored() {
	value := &Local{Value: 1}
	holder := Stored{Pointer: value}
	_ = holder
}
