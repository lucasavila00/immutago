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
	Lookup(key, buf []byte) (float64, bool, []byte, error)
	Read() (im.StringProjection, bool)
	SummaryLookup(Buffer) (Summary, bool, Buffer, Failure)
	Clone() []byte
	Self() Reader
	Ordinary(Input) (string, any, func() int, map[int]int, [][]byte, error)
}

type Alias = Scorer

type Summary struct {
	im.Copied
	Count int
}

type Buffer []byte
type Bytes = Buffer
type Failure = error

type Input struct {
	Value    any
	Callback func() int
}
