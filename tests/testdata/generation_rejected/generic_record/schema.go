package generic_record

//go:generate go run github.com/lucasavila00/immutago/cmd/immutago-gen

import (
	"github.com/lucasavila00/immutago/pkg/im"
)

type Box[T ~int | ~string] struct {
	im.Frozen
	Value T
	Next  *Box[T]
}
