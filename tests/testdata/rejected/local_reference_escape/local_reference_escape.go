package local_reference_escape

import "github.com/lucasavila00/immutago/pkg/im"

type Local struct {
	im.LocalRef
	Value int
}
type Stored struct {
	im.Shared
	Pointer *Local
}

var global *Local

func returned() *Local      { return &Local{Value: 1} }
func retained(value *Local) { global = value }
func stored() {
	value := &Local{Value: 1}
	holder := Stored{Pointer: value}
	_ = holder
}
func collected() {
	value := &Local{Value: 1}
	builder := []*Local{value}
	frozen := im.FreezeSlice(builder)
	_ = frozen
}
func captured() {
	value := &Local{Value: 1}
	callback := func() int { return value.Value }
	_ = callback
}
func writes() {
	value := Local{Value: 1}
	value.Value = 2
	pointer := &Local{Value: 1}
	pointer.Value = 2
}

func readLocal(value *Local) int { return value.Value }
func delayed() {
	value := &Local{Value: 1}
	go readLocal(value)
	defer readLocal(value)
}

type Both struct {
	im.Shared
	im.LocalRef
	Value int
}

type LocalStored struct {
	im.LocalRef
	Pointer *Local
}
