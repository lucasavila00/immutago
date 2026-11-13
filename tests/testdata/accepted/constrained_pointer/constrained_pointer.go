package rejected

import "github.com/lucasavila00/immutago/pkg/im"

type Good struct {
	im.Marker
	N int
}

func replace[T Good](p *T) {}
