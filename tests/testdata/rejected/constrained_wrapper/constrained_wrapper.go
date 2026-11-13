package rejected

import im "github.com/lucasavila00/immutago/pkg/immutable"

type Good struct {
	im.Marker
	N int
}
type Bad[T Good] struct{ Value T } // want "unmarked struct contains immutable field"
