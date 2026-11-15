package model

import "github.com/lucasavila00/immutago/pkg/im"

type Model struct {
	im.Frozen
	Version string
	Weights im.FrozenSlice[float64]
}

type Frozen struct {
	im.Frozen
	N int
}

type SafeCell = im.Cell[Model]
type Scalar[T interface{ ~int | SafeCell }] = int
