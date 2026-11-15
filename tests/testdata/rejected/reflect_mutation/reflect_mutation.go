//go:generate go run github.com/lucasavila00/immutago/cmd/immutago-gen

package rejected

import (
	"github.com/lucasavila00/immutago/pkg/im"
	"reflect"
)

type Value struct {
	im.Frozen
	N int
}

func mutate() { reflect.ValueOf([]Value{{N: 1}}).Index(0).FieldByName("N").SetInt(2) }
