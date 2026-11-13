package rejected

import im "github.com/lucasavila00/immutago/pkg/immutable"

type Good struct {
	im.Marker
	Value int
	Array [2]int
}

func mutate() {
	var value Good
	for value.Value = range []int{1} { // want "cannot mutate a field"
	}
}
