package rejected

import "github.com/lucasavila00/immutago/pkg/im"

type Count int

func (p *Count) Change() { *p = 1 }

type Value struct {
	im.Shared
	N Count
}

func mutate() {
	var value Value
	pointer := new(value)
	value.N.Change()
	_ = pointer.N
}
