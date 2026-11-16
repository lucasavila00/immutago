package generic_pointer_receiver

//go:generate go run github.com/lucasavila00/immutago/cmd/immutago-gen

import (
	"github.com/lucasavila00/immutago/pkg/im"
)

type Box[T ~int] struct {
	im.Frozen
	X T
}

func (b *Box[T]) Read() T { return b.X }
func f()                  { var b Box[int]; _ = b.Read() }
