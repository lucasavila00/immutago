package unchecked

import (
	"sync"

	"github.com/lucasavila00/immutago/pkg/im"
)

// baked is an internal structure that the package trusts itself never to change after Load.
type baked struct {
	mu      sync.Mutex
	weights []float64
}

// Model is an immutable object with a trusted internal part.
type Model struct {
	im.Shared
	Name   string
	engine im.Unchecked[baked]
}

// Load builds a model and trusts its internal part.
func Load(name string, weights []float64) *Model {
	return &Model{Name: name, engine: im.Trust(&baked{weights: weights})}
}

// Predict reads the internal part through the API of the declaring package.
func Predict(m *Model, row int) float64 {
	b := m.engine.Value()
	return b.weights[row]
}
