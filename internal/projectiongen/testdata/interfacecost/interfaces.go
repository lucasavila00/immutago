//go:generate go run github.com/lucasavila00/immutago/cmd/immutago-gen

package interfacecost

import "github.com/lucasavila00/immutago/pkg/im"

type Scorer interface {
	im.FrozenInterface
	Score() int
	Lookup([]byte) (int, bool, []byte, error)
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

func (n *Number) Lookup(buf []byte) (int, bool, []byte, error) { return n.N, true, buf[:1], nil }

var buffer [1]byte
var output []byte
var found bool
var failure error

func readResults() {
	value, ok, out, err := current.Load().Value().Lookup(buffer[:])
	result = value
	found = ok
	output = out
	failure = err
}
