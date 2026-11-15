//go:generate go run github.com/lucasavila00/immutago/cmd/immutago-gen

package rejected

import "github.com/lucasavila00/immutago/pkg/im"

type Good struct {
	im.Frozen
	Value int
	Array [2]int
}
type Outer struct {
	im.Frozen
	Child Good
}

func mutate() { var value Outer; value.Child.Array[0] = 1 }
