package model_test

import (
	"testing"

	"github.com/lucasavila00/immutago/examples/models/model"
	"github.com/lucasavila00/immutago/pkg/im"
)

// TestLoadCopiesWeights checks that later caller writes cannot change a prediction.
func TestLoadCopiesWeights(t *testing.T) {
	weights := []float64{2, 3, -1}
	m := model.Load("score", weights)
	weights[0] = 100
	features := im.FreezeSlice([]float64{2, 3, 4})
	if got := model.Predict(m, features); got != 9 {
		t.Fatalf("prediction after input change = %v, want 9", got)
	}
	if got := m.Name.Clone(); got != "score" {
		t.Fatalf("model name = %q", got)
	}
}
