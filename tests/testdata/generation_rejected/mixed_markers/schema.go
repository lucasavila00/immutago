package mixed_markers

//go:generate go run github.com/lucasavila00/immutago/cmd/immutago-gen

import (
	"github.com/lucasavila00/immutago/pkg/im"
)

type Both struct {
	im.Frozen
	im.Local
	Value int
}
