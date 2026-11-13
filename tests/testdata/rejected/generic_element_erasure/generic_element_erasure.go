package rejected

import "github.com/lucasavila00/immutago/pkg/im"

type Good struct {
	im.Marker
	Value int
}

func head[T any](values []T) any { return &values[0] } // want "unsupported pointer to unresolved generic storage"
var Hidden = head([]Good{{Value: 1}})
