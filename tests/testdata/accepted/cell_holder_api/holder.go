package cell_holder_api

import "github.com/lucasavila00/immutago/pkg/im"

type Holder struct {
	Slot  im.Cell[int]
	Calls int
}

type Alias = Holder
type Pointer *Holder
type Nested struct{ Child Holder }
type Parent struct{ Child *Holder }
type Embedded struct{ *Holder }
type Callback func(...*Holder) *Holder
type Store interface{ Publish(int) }
type Box[T any] struct{ Value T }

var Current = New()

func New() *Holder                   { return &Holder{Calls: 1} }
func (h *Holder) Publish(n int)      { h.Slot.Store(n) }
func (h *Holder) Read() int          { return im.Value(h.Slot.Load()) }
func (h *Holder) Count() int         { return h.Calls }
func (h *Holder) Increment()         { h.Calls++ }
func Child(n *Nested) *Holder        { return &n.Child }
func Identity[T any](value T) T      { return value }
func Same[T ~*Holder](value T) T     { return value }
func Factory[T any](value T) *Holder { return new(Holder) }

func Local() *Holder {
	var value Holder
	return &value
}

func Share(h *Holder) (Parent, *Holder) { return Parent{Child: h}, h }

func Retain(values ...*Holder) *Holder { return values[0] }

func Closure(h *Holder) func() *Holder {
	return func() *Holder {
		h.Publish(4)
		return h
	}
}

func CapturePointer() func() int {
	var h Holder
	pointer := &h
	return func() int { return pointer.Calls }
}

func CaptureVariable() (func(int), func() int) {
	var h Holder
	publish := func(value int) { h.Publish(value) }
	read := func() int { return h.Read() }
	return publish, read
}

func CaptureScalarField() func() int {
	var h Holder
	return func() int { return h.Calls }
}
