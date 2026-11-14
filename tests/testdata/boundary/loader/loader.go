package loader

import (
	"example.com/consumer/logx"
	"example.com/consumer/model"
	"github.com/lucasavila00/immutago/pkg/im"
)

type Loader struct{ Current im.Owner[model.Model] }

func Publish(l *Loader, version string, weights []float64) {
	values := make([]float64, len(weights))
	copy(values, weights)
	frozen := im.FreezeSlice(values)
	l.Current.Store(&model.Model{Version: version, Weights: frozen})
	logx.Info("published", len(weights))
}

func LogVersion(l *Loader) {
	current := l.Current.Load()
	logx.Info(current.Version, current.Weights.Len())
}
