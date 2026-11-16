package interface_isnil

//go:generate go run github.com/lucasavila00/immutago/cmd/immutago-gen

import "github.com/lucasavila00/immutago/pkg/im"

type Record interface {
	im.FrozenInterface
	IsNil() bool
}
