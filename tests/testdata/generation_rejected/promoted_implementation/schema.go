package promoted_implementation

//go:generate go run github.com/lucasavila00/immutago/cmd/immutago-gen

import (
	"github.com/lucasavila00/immutago/pkg/im"
)

type Scorer interface {
	im.FrozenInterface
	Score([]float32) float64
}
type Mutable struct {
	im.Frozen
	Values im.FrozenSlice[int]
}

func (m *Mutable) Score([]float32) float64 { return 0 }

type Promoted struct {
	im.Frozen
	*Mutable
}

func PromotedValue() Scorer { return Promoted{Mutable: &Mutable{}} }
