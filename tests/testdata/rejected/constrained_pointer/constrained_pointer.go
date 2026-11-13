package rejected

import im "github.com/lucasavila00/immutago/pkg/immutable"

type Good struct {
	im.Marker
	N int
}

func replace[T Good](p *T) {} // want "unsupported pointer to immutable storage"
