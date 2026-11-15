package rejected

import "github.com/lucasavila00/immutago/pkg/im"

type Count struct {
	im.Snapshot
	Values [2]int
}

func (p *Count) Change() { p.Values[0] = 1 }

type Value struct {
	im.Snapshot
	N Count
}

func mutate() {
	var value Value
	pointer := new(value)
	value.N.Change()
	_ = pointer.N
}
