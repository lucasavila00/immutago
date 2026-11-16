package generic_receiver

//go:generate go run github.com/lucasavila00/immutago/cmd/immutago-gen

import (
	"github.com/lucasavila00/immutago/pkg/im"
)

type Generic[T any] struct {
	im.Frozen
	Value T
}

func (g Generic[T]) Read() T { return g.Value }
