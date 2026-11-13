package rejected

import "github.com/lucasavila00/immutago/pkg/im"

type Good struct {
	im.Immut
	Value int
	Array [2]int
}
type Derived Good

func mutate() { var value Derived; value.Value = 1 }
