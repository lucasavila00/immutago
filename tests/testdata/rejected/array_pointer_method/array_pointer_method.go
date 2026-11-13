package rejected

import im "github.com/lucasavila00/immutago/pkg/im"

type Count struct {
	im.Marker
	Values [2]int
}

func (p *Count) Change() { p.Values[0] = 1 }

type Value struct {
	im.Marker
	N Count
}

func mutate() {
	var value Value
	pointer := &value
	value.N.Change() // want "borrow|shared reference"
	_ = pointer.N
}
