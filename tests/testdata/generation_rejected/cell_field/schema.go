package cell_field

//go:generate go run github.com/lucasavila00/immutago/cmd/immutago-gen

import (
	"github.com/lucasavila00/immutago/pkg/im"
)

type Model struct {
	im.Frozen
	X int
}
type BadCell struct {
	im.Frozen
	Current im.Cell[Model]
}
