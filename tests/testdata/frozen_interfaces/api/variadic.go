package api

import "github.com/lucasavila00/immutago/pkg/im"

type Amount int
type AmountAlias = Amount
type VariadicReader interface {
	im.FrozenInterface
	Sum(...int) int
	Retain(...int)
	Add(prefix int, values ...AmountAlias) int
	Call(...func() int) int
	Box(...any) int
}

type VariadicAlias = VariadicReader
type Counts []int
