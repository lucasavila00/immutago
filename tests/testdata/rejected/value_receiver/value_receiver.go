package rejected

import "github.com/lucasavila00/immutago/pkg/im"

type Good struct {
	im.Marker
	Value int
	Array [2]int
}

func (value Good) Mutate() { value.Value = 1 } // want "cannot mutate a field"
