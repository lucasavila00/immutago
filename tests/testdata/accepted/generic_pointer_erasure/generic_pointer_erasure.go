package rejected

import "github.com/lucasavila00/immutago/pkg/im"

type Good struct {
	im.Marker
	Value int
}

func box[T any](value T) any { return &value }

var Hidden = box(Good{Value: 1})
