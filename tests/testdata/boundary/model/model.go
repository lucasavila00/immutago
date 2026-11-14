package model

import "github.com/lucasavila00/immutago/pkg/im"

type Model struct {
	im.Scoped
	Version string
	Weights im.FrozenSlice[float64]
}

type Shared struct {
	im.Shared
	N int
}
