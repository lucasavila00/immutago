package rejected

import im "github.com/lucasavila00/immutago/pkg/im"

type Count int

func (p *Count) Change() { *p = 1 }

type Value struct {
	im.Marker
	N Count
}

func mutate() {
	var value Value
	pointer := &value
	method := value.N.Change
	method() // want "borrow|shared reference"
	_ = pointer.N
}
