package rejected

import "github.com/lucasavila00/immutago/pkg/im"

type Good struct {
	im.Shared
	Value int
	Array [2]int
}
type Outer struct {
	im.Shared
	Child Good
}

func mutate() { var value Outer; value.Child.Array[0] = 1 }
