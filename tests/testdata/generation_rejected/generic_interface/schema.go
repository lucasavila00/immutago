package generic_interface

//go:generate go run github.com/lucasavila00/immutago/cmd/immutago-gen

import "github.com/lucasavila00/immutago/pkg/im"

type Bad[T any] interface {
	im.FrozenInterface
	Read() T
}
