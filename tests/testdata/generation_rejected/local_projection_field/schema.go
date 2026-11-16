package local_projection_field

//go:generate go run github.com/lucasavila00/immutago/cmd/immutago-gen

import (
	"github.com/lucasavila00/immutago/pkg/im"
)

type Local struct {
	im.Local
	Text im.StringProjection
}
type BadSnapshot struct {
	im.Frozen
	Result Local
}
