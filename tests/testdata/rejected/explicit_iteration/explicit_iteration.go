package explicit_iteration

import "github.com/lucasavila00/immutago/pkg/im"

type Good struct {
	im.Immut
	X int
}

func pointerIterator(yield func(*Good) bool) {}
func valueIterator(yield func(Good) bool)    {}
func pointerRange() {
	for pointer := range pointerIterator {
		_ = pointer.X
	}
}
func valueRange() {
	for value := range valueIterator {
		_ = value.X
	}
}
