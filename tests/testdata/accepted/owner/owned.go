package owner

import "github.com/lucasavila00/immutago/pkg/im"

type Model struct {
	im.Scoped
	Version string
	Weights im.FrozenSlice[float64]
}

type Loader struct{ Current im.Owner[Model] }

var Current im.Owner[Model]

var current im.Owner[Model]

func Publish(l *Loader, version string, weights []float64) {
	w := make([]float64, len(weights))
	copy(w, weights)
	frozen := im.FreezeSlice(w)
	l.Current.Store(&Model{Version: version, Weights: frozen})
}

func Handle(l *Loader, features []float64) float64 {
	m := l.Current.Load()
	return Score(m, features)
}

func Score(m *Model, features []float64) float64 {
	total := 0.0
	for i, w := range m.Weights.All() {
		total += w * features[i]
	}
	return total
}

func Global() {
	current.Store(new(Model))
	m := current.Load()
	_ = m
}

// Background starts a checked publisher with a direct holder argument.
func Background(l *Loader) { go Publish(l, "next", nil) }

// NewLoader uses the zero holder allocation form.
func NewLoader() {
	l := new(Loader)
	l.Current.Store(new(Model))
}
