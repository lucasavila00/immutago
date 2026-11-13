package rejected

import (
	"github.com/lucasavila00/immutago/pkg/im"
	"reflect"
)

type Value struct {
	im.Immut
	N int
}

func mutate() { reflect.ValueOf([]Value{{N: 1}}).Index(0).FieldByName("N").SetInt(2) }
