package rejected

import (
	im "github.com/lucasavila00/immutago/pkg/immutable"
	"reflect" // want "unsupported reflect import"
)

type Value struct {
	im.Marker
	N int
}

func mutate() { reflect.ValueOf([]Value{{N: 1}}).Index(0).FieldByName("N").SetInt(2) }
