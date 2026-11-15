package api

import "github.com/lucasavila00/immutago/pkg/im"

type InputReader interface {
	im.FrozenInterface
	Apply(value any, callback func() int) int
	Named(NumberReader, Callback) int
	Aliased(ReaderAlias, CallbackAlias) int
	ReadLocal(im.StringProjection) int
	WithScorer(Scorer) int
}

type NumberReader interface{ Number() int }
type Callback func() int
type ReaderAlias = NumberReader
type CallbackAlias = Callback
