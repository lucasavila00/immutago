package private_interface

//go:generate go run github.com/lucasavila00/immutago/cmd/immutago-gen

import (
	"github.com/lucasavila00/immutago/pkg/im"
)

type Private interface {
	im.FrozenInterface
	read() int
}
