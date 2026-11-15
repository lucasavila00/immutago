//go:generate go run github.com/lucasavila00/immutago/cmd/immutago-gen

// Package model supplies immutable linear prediction models.
package model

import (
	"slices"

	"github.com/lucasavila00/immutago/pkg/im"
)

// baked owns coefficients that this package never changes after Load.
type baked struct {
	weights []float64
}

// Model holds a name and a private immutable copy of its coefficients.
type Model struct {
	im.Frozen
	Name   string
	engine im.UncheckedValue[baked]
}

// Load copies weights so the caller can continue to use its input slice.
func Load(name string, weights []float64) *Model {
	return &Model{Name: name, engine: im.Trust(&baked{weights: slices.Clone(weights)})}
}

// Predict returns the dot product of features and the model's coefficients.
// The feature count must equal the coefficient count.
func Predict(m *Model, features im.FrozenSlice[float64]) float64 {
	b := im.Unwrap(m.engine)
	if features.Len() != len(b.weights) {
		panic("feature count does not match model")
	}
	var score float64
	for i, weight := range b.weights {
		score += weight * features.At(i)
	}
	return score
}
