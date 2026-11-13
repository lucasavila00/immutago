package rejected

import "github.com/lucasavila00/immutago/pkg/im"

type Good struct {
	im.Immut
	Value int
	Array [2]int
}
type Outer struct {
	im.Immut
	Child Good
}

func mutate() { var value Outer; value.Child.Array[0] = 1 }
