package rejected

import (
	"example.com/consumer/api"
	"github.com/lucasavila00/immutago/pkg/im"
)

var typedCell api.ModelCell

type Embedded struct{ api.ModelCell }
type Array struct{ Cells [2]api.ModelCell }

type SlotConstraint interface{ im.Cell[int] }
type ConstraintAlias = SlotConstraint

func copySlot[T ConstraintAlias](value T) T { return value }

type DefinedPointer *Data

func holderForms(h *api.Holder, pointer *api.ModelCell, embedded *Embedded) {
	copy := h.Current
	_ = copy
	h.Current = api.ModelCell{}
	address := &h.Current
	_ = address
	read := h.Current.Load
	_ = read
	_ = (*api.ModelCell).Load(pointer)
	_ = embedded.Load()
	go typedCell.Count()
	defer typedCell.Count()
	go typedCell.Load()
	defer typedCell.Load()
}
