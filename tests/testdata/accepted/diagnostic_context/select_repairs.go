package diagnosticcontext

import "github.com/lucasavila00/immutago/pkg/im"

func selectReads(pointers chan<- *Settings, numbers chan<- int, ready <-chan struct{},
	view SettingsProjection) im.FrozenSlice[int] {
	limit := view.Limit()
	pointer := view.Unchecked()
	currentScorer.Store(pointer)
	values := make([]int, 1)
	select {
	case pointers <- pointer:
	case numbers <- limit:
	case <-ready:
		values[0] = view.Limit()
		_ = currentScorer.Load()
	}
	frozen := im.FreezeSlice(values)
	return frozen
}
