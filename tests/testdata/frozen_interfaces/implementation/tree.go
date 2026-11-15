//go:generate go run github.com/lucasavila00/immutago/cmd/immutago-gen

package implementation

import (
	"example.com/consumer/api"
	"github.com/lucasavila00/immutago/pkg/im"
)

type Tree struct {
	im.Frozen
	Weight float64
	Text   string
	Data   im.FrozenSlice[byte]
}

func (t *Tree) Score(features []float32) float64 { return t.Weight * float64(len(features)) }
func (t *Tree) Name() im.StringProjection        { return im.ProjectString(t.Text) }

var Saved api.Scorer

// Visit can retain its own Frozen receiver under the ordinary method contract.
func (t *Tree) Visit() { Saved = t }

func Make() api.Reader {
	values := []byte{4, 5}
	frozen := im.FreezeSlice(values)
	return &Tree{Weight: 3, Text: "tree", Data: frozen}
}

type Count struct {
	im.Frozen
	N int
}

func (c Count) Score([]float32) float64 { return float64(c.N) }

func (t *Tree) Summary() api.Summary { return api.Summary{Count: int(t.Weight)} }

// Lookup returns the caller buffer with the ordinary Go slice contract.
func (t *Tree) Lookup(key, buf []byte) (float64, bool, []byte, error) {
	buf = append(buf[:0], key...)
	return t.Weight, len(key) != 0, buf, nil
}

func (t *Tree) Read() (im.StringProjection, bool) { return im.ProjectString(t.Text), true }
func (t *Tree) SummaryLookup(buf api.Buffer) (api.Summary, bool, api.Buffer, api.Failure) {
	return t.Summary(), true, buf[:0], nil
}
func (t *Tree) Clone() []byte { return t.Data.Clone() }

func (t *Tree) Self() api.Reader { return t }
func (t *Tree) Ordinary(input api.Input) (string, any, func() int, map[int]int, [][]byte, error) {
	return t.Text, input.Value, input.Callback, map[int]int{1: 2}, [][]byte{nil}, nil
}

type TreeReader interface {
	im.FrozenInterface
	TreePointer() *Tree
}

func (t *Tree) TreePointer() *Tree { return t }
