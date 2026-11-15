//go:generate go run github.com/lucasavila00/immutago/cmd/immutago-gen

package interfacecost

import "github.com/lucasavila00/immutago/pkg/im"

type Scorer interface {
	im.FrozenInterface
	Score() int
}

type Number struct {
	im.Frozen
	N int
}

func (n *Number) Score() int { return n.N }

type Model struct {
	im.Frozen
	Value Scorer
}

var current ModelCell
var result int

func read() { result = current.Load().Value().Score() }
