package localfield

//go:generate go run github.com/lucasavila00/immutago/cmd/immutago-gen

import "github.com/lucasavila00/immutago/pkg/im"

type Settings struct {
	im.Frozen
	Limit int
	Title im.StringProjection
}
