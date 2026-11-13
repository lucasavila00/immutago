package rejected

import "github.com/lucasavila00/immutago/pkg/im"

type Good struct {
	im.Marker
	Value int
}

func box[T any](value T) any { return &value } // want "unsupported pointer to unresolved generic storage"
var Hidden = box(Good{Value: 1})
