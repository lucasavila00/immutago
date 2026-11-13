package rejected

import "github.com/lucasavila00/immutago/pkg/im"

type Good struct {
	im.Immut
	Value int
	Array [2]int
}
type Wrapper[T any] struct{ Value T }

var Bad Wrapper[Good]
