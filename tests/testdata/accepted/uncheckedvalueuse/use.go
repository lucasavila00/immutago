//go:generate go run github.com/lucasavila00/immutago/cmd/immutago-gen

package uncheckedvalueuse

import (
	"example.com/consumer/uncheckedvalue"
	"github.com/lucasavila00/immutago/pkg/im"
)

// Current is the published model.
type Current struct {
	im.Frozen
	Model *uncheckedvalue.Model
}

// Loader publishes models.
type Loader struct {
	current im.Cell[Current]
}

// Publish stores a new model and predicts with the current one.
func Publish(l *Loader, weights []float64) float64 {
	l.current.Store(Current{Model: uncheckedvalue.Load("next", weights)})
	current := l.current.Shared()
	model := current.Model
	return uncheckedvalue.Predict(model, 0)
}
