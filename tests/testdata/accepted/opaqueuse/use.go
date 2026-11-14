package opaqueuse

import (
	"example.com/consumer/opaque"
	"github.com/lucasavila00/immutago/pkg/im"
)

// Current is the published model.
type Current struct {
	im.Scoped
	Model *opaque.Model
}

// Loader publishes models.
type Loader struct {
	current im.Owner[Current]
}

// Publish stores a new model and predicts with the current one.
func Publish(l *Loader, weights []float64) float64 {
	l.current.Store(&Current{Model: opaque.Load("next", weights)})
	current := l.current.Load()
	model := current.Model
	return opaque.Predict(model, 0)
}
