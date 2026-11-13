package rejected

import im "github.com/lucasavila00/immutago/pkg/immutable"

type Good struct {
	im.Marker
	Value int
}

func box[T any](value T) any { return &value } // want "unsupported pointer to unresolved generic storage"
var Hidden = box(Good{Value: 1})
