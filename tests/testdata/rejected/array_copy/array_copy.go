package rejected

import "github.com/lucasavila00/immutago/pkg/im"

type Good struct {
	im.Marker
	Value int
	Array [2]int
}

func mutate() { var value Good; copy(value.Array[:], []int{1}) } // want "cannot mutate a field"
