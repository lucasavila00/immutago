package rejected

import "github.com/lucasavila00/immutago/pkg/im"

type Count struct {
	im.Shared
	Values [2]int
}

func (p *Count) Change() { p.Values[0] = 1 }

type Value struct {
	im.Shared
	N Count
}

func mutate() {
	var value Value
	pointer := new(value)
	method := value.N.Change
	method()
	_ = pointer.N
}
