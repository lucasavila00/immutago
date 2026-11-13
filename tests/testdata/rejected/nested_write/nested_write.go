package rejected

import im "github.com/lucasavila00/immutago/pkg/immutable"

type Good struct {
	im.Marker
	Value int
	Array [2]int
}
type Outer struct {
	im.Marker
	Child Good
}

func mutate() { var value Outer; value.Child.Array[0] = 1 } // want "cannot mutate a field"
