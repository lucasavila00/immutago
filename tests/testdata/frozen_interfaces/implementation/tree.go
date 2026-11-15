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
}

func (t *Tree) Score(features []float32) float64 { return t.Weight * float64(len(features)) }
func (t *Tree) Name() im.StringProjection        { return im.ProjectString(t.Text) }

var Saved api.Scorer

// Visit can retain its own Frozen receiver under the ordinary method contract.
func (t *Tree) Visit() { Saved = t }

func Make() api.Reader { return &Tree{Weight: 3, Text: "tree"} }

type Count struct {
	im.Frozen
	N int
}

func (c Count) Score([]float32) float64 { return float64(c.N) }

func (t *Tree) Summary() api.Summary { return api.Summary{Count: int(t.Weight)} }
