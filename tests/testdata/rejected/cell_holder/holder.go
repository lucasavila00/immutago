package cell_holder

import "github.com/lucasavila00/immutago/pkg/im"

type Holder struct {
	Slot im.Cell[int]
	N    int
}
type Alias = Holder
type Nested struct{ Child Holder }
type Embedded struct{ *Holder }
type BadEmbedded struct{ Holder }
type BadArray struct{ Values [1]Holder }
type BadRawPointer struct{ Slot *im.Cell[int] }
type BadLocal struct{ View im.Projection[int] }
type Generic[T any] struct{ Slot im.Cell[int] }

var saved im.Projection[int]

func (h *Holder) View() im.Projection[int] { return h.Slot.Load() }
func (h Holder) Copy() int                 { return h.N }
func ValueParameter(Holder)                {}
func ValueResult(h *Holder) Holder         { return *h }
func TupleResult(h *Holder) (int, Holder)  { return 0, *h }

func CopyValues(h *Holder, nested *Nested) {
	copy := *h
	_ = copy
	var alias Alias = *h
	_ = alias
	var boxed any = *h
	_ = boxed
	*h = Holder{}
	*h = *h
	h.Slot = im.Cell[int]{}
	nested.Child = Holder{}
	_, nested.Child = 0, Holder{}
	_, *h = 0, Holder{}
	_ = Nested{Child: *h}
	_ = &Holder{Slot: h.Slot}
	_, _ = TupleResult(h)
}

func ImplicitCopies(h *Holder, e Embedded) {
	_ = h.Copy()
	call := h.Copy
	_ = call()
	_ = (*Holder).Copy(h)
	_ = e.Copy()
}

func PointerHelper[T any](p *T) *T { return p }

func GenericCopy[T ~*Holder](p T) { _ = *p }

func GenericCall(h *Holder) { _ = PointerHelper(h) }

func RawSlots(h *Holder, e Embedded) {
	pointer := &h.Slot
	_ = pointer
	_ = new(im.Cell[int])
	_ = e.Slot.Load()
	load := h.Slot.Load
	_ = load
	go h.Slot.Store(1)
	defer h.Slot.Store(2)
}

func LocalResults(h *Holder) {
	saved = h.View()
	view := h.View()
	_ = func() { _ = view }
	read := h.View
	_ = read
	go h.View()
}
