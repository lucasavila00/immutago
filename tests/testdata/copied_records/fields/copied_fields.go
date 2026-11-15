package fields

import (
	"example.com/consumer/api"
	"github.com/lucasavila00/immutago/pkg/im"
	"github.com/lucasavila00/immutago/pkg/supervise"
)

type Worker struct {
	supervise.Service[supervise.NoArgs]
}
type BadString struct {
	im.Copied
	Value string
}
type BadFrozenString struct {
	im.Copied
	Value im.FrozenString
}
type BadPointer struct {
	im.Copied
	Value *api.Point
}
type BadSlice struct {
	im.Copied
	Value []int
}
type BadMap struct {
	im.Copied
	Value map[int]int
}
type BadChannel struct {
	im.Copied
	Value chan int
}
type BadFunction struct {
	im.Copied
	Value func()
}
type BadInterface struct {
	im.Copied
	Value any
}
type BadZeroArray struct {
	im.Copied
	Value [0]string
}
type BadNestedArray struct {
	im.Copied
	Value [2][0]*api.Point
}
type BadRecord struct {
	im.Copied
	Value struct{ N int }
}
type BadReference struct {
	im.Copied
	Value supervise.Ref[Worker]
}
type BadReferenceArray struct {
	im.Copied
	Value [0]supervise.Ref[Worker]
}
type BadCollection struct {
	im.Copied
	Value im.FrozenSlice[int]
}
type BadProjection struct {
	im.Copied
	Value im.Projection[int]
}
type BadCell struct {
	im.Copied
	Value im.Cell[int]
}
type BadTrust struct {
	im.Copied
	Value im.UncheckedValue[api.Point]
}
type BadMarker struct {
	im.Copied
	Value [0]im.Copied
}
type Array [2]int
type BadNamedArray struct {
	im.Copied
	Value Array
}
type BadGeneric[T any] struct {
	im.Copied
	Value T
}
type BadNamedMarker struct{ Marker im.Copied }
type BadPointerMarker struct{ *im.Copied }
type MarkerAlias = im.Copied
type BadDoubleMarker struct {
	im.Copied
	MarkerAlias
}
type BadLocalMarker struct {
	im.Copied
	im.Local
}

func (*Worker) Bad() *api.Point           { return nil }
func (*Worker) Unmarked() struct{ N int } { return struct{ N int }{} }
func (*Worker) Marker() im.Copied         { return im.Copied{} }

func (*Worker) Pair() (api.Point, bool) { return api.Point{}, true }
