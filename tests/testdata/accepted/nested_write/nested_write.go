package rejected

import "github.com/lucasavila00/immutago/pkg/im"

type Good struct {
	im.Marker
	Value int
	Array [2]int
}
type Outer struct {
	im.Marker
	Child Good
}

func mutate() { var value Outer; value.Child.Array[0] = 1 }
