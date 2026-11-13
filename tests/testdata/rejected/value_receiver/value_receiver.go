package rejected

import im "github.com/lucasavila00/immutago/pkg/immutable"

type Good struct {
	im.Marker
	Value int
	Array [2]int
}

func (value Good) Mutate() { value.Value = 1 } // want "cannot mutate a field"
