//go:generate go run github.com/lucasavila00/immutago/cmd/immutago-gen

package invalid

import (
	"example.com/consumer/api"
	"github.com/lucasavila00/immutago/pkg/im"
)

type Mutable struct {
	im.Frozen
	Values []int
}

func (m *Mutable) Score([]float32) float64 { return 0 }
func Make() api.Scorer                     { return &Mutable{} }

type Hidden struct{ api.Scorer }

func HiddenValue(s api.Scorer) api.Scorer { return Hidden{Scorer: s} }

type Promoted struct {
	im.Frozen
	*Mutable
}

func PromotedValue() api.Scorer { return Promoted{Mutable: &Mutable{}} }
