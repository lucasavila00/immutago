package owner

import "github.com/lucasavila00/immutago/pkg/im"

type Model struct {
	im.Frozen
	Version string
	Weights im.FrozenSlice[float64]
}

type Loader struct{ Current im.Cell[Model] }

var Current im.Cell[Model]

var current im.Cell[Model]

func Publish(l *Loader, version string, weights []float64) {
	w := make([]float64, len(weights))
	copy(w, weights)
	frozen := im.FreezeSlice(w)
	l.Current.Store(Model{Version: version, Weights: frozen})
}

func Handle(l *Loader, features []float64) float64 {
	m := l.Current.Load()
	return Score(m, features)
}

func Score(m im.View[Model], features []float64) float64 {
	total := 0.0
	weights := im.Slice(im.Select(m, func(value Model) im.FrozenSlice[float64] { return value.Weights }))
	for i, w := range weights.All() {
		total += im.Copy(w) * features[i]
	}
	return total
}

func Global() {
	current.Store(Model{})
	m := current.Load()
	_ = m
}

// Background starts a checked publisher with a direct holder argument.
func Background(l *Loader) { go Publish(l, "next", nil) }

// NewLoader uses the zero holder allocation form.
func NewLoader() {
	l := new(Loader)
	l.Current.Store(Model{})
}

// Republish stores an existing snapshot through an independent root copy.
func Republish(l *Loader) {
	value := l.Current.Shared()
	l.Current.Store(value)
}
