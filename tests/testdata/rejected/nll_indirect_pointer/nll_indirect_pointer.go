package rejected

import im "github.com/lucasavila00/immutago/pkg/im"

type Good struct {
	im.Marker
	X int
}

func dereference(pointer **Good) *Good { return *pointer }
func read(pointer *Good) int           { return pointer.X }
func bad() {
	var value Good
	pointer := &value
	alias := dereference(&pointer)
	value.X = 2
	_ = read(alias)
} // want "borrow|shared reference|unsupported"
