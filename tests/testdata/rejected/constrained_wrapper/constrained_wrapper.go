package rejected

import "github.com/lucasavila00/immutago/pkg/im"

type Good struct {
	im.Marker
	N int
}
type Bad[T Good] struct{ Value T }
