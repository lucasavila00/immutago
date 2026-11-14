package loader

import (
	"example.com/consumer/logx"
	"github.com/lucasavila00/immutago/pkg/im"
)

func Leak() {
	values := []float64{1}
	frozen := im.FreezeSlice(values)
	logx.Info("weights", frozen)
}
