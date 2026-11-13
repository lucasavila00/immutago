package rejected

import im "github.com/lucasavila00/immutago/pkg/immutable"

type Count int

func (p *Count) Change() { *p = 1 }

type Value struct {
	im.Marker
	N Count
}

func mutate() { var value Value; value.N.Change() } // want "cannot mutate a field"
