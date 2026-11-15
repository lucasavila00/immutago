package rejected

import "example.com/consumer/api"

var typedCell api.ModelCell

type Embedded struct{ api.ModelCell }
type Array struct{ Cells [2]api.ModelCell }

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
