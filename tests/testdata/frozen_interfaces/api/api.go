//go:generate go run github.com/lucasavila00/immutago/cmd/immutago-gen

package api

import "github.com/lucasavila00/immutago/pkg/im"

type Scorer interface {
	im.FrozenInterface
	Score(features []float32) float64
}

type Reader interface {
	Scorer
	Name() im.StringProjection
	Summary() Summary
	Visit()
}

type Alias = Scorer

type Summary struct {
	im.Copied
	Count int
}
