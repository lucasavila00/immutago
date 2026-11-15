package accepted

import "example.com/consumer/api"

type SharedHolder struct{ Current *api.ModelCell }
type EmbeddedHolder struct{ *api.ModelCell }

var Shared = SharedHolder{Current: new(api.ModelCell)}

func NewHolder() *api.ModelCell { return new(api.ModelCell) }

func NestedHolder(h *api.Holder) *api.ModelCell { return &h.Current }

func ReadShared(h *api.ModelCell) api.ModelProjection {
	embedded := EmbeddedHolder{h}
	view := embedded.Load()
	return view
}

func CountShared(h *api.ModelCell) {
	count := h.Count
	_ = count()
	_ = (*api.ModelCell).Count(h)
	go h.Count()
	defer h.Count()
}
