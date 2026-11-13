package rejected

import im "github.com/lucasavila00/immutago/pkg/immutable"

type Good struct {
	im.Marker
	Value int
	Array [2]int
}
type Wrapper[T any] struct{ Value T }

var Bad Wrapper[Good] // want "unmarked struct contains immutable field"
