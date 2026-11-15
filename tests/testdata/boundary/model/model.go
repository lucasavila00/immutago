package model

import "github.com/lucasavila00/immutago/pkg/im"

type Model struct {
	im.Snapshot
	Version string
	Weights im.FrozenSlice[float64]
}

type Snapshot struct {
	im.Snapshot
	N int
}

type SafeCell = im.Cell[Model]
type Scalar[T interface{ ~int | SafeCell }] = int
