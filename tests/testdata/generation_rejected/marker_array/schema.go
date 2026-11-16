package marker_array

//go:generate go run github.com/lucasavila00/immutago/cmd/immutago-gen

import (
	"github.com/lucasavila00/immutago/pkg/im"
)

type BadMarkers struct {
	im.Frozen
	Values [2]im.Frozen
}
