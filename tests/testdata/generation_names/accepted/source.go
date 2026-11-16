package accepted

//go:generate go run github.com/lucasavila00/immutago/cmd/immutago-gen

import (
	"example.com/consumer/api"
	immutable "github.com/lucasavila00/immutago/pkg/im"
)

type im int

var im2 int

func immutagoLocal() {}

const immutagoLocal2 = 2

type immutagoLocal3 int
type Marker = immutable.Frozen
type Record struct {
	Marker
	Local               int
	immutagoLocal4      int
	immutagoProjection  int
	immutagoProjection2 int
	Grid                [2][3]int
	Reader              Reader
	Foreign             api.Alias
	Pointer             api.Pointer
	ForeignArray        [2]api.Alias
	PointerArray        [2]api.Pointer
}
type Reader interface {
	immutable.FrozenInterface
	Local() int
	Unsafe() int
}
type Number struct {
	immutable.Frozen
	N int
}

func (n *Number) Local() int  { return n.N }
func (n *Number) Unsafe() int { return n.N + 1 }

var current RecordCell

func Read() int {
	current.Store(Record{Local: 1, immutagoLocal4: 2, immutagoProjection: 3,
		immutagoProjection2: 4, Grid: [2][3]int{{}, {0, 0, 5}}, Reader: &Number{N: 6}})
	p := current.Load()
	return p.Local() + p.immutagoLocal4() + p.immutagoProjection() + p.immutagoProjection2() +
		p.Grid().At(1).At(2) + p.Reader().Local() + p.Reader().Unsafe() + p.Foreign().N()
}
