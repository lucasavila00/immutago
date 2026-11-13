package rejected

import im "github.com/lucasavila00/immutago/pkg/immutable"

type Numbers [2]int

func (p *Numbers) Change() { p[0] = 1 }

type Value struct {
	im.Marker
	N Numbers
}

func mutate() { var value Value; value.N.Change() } // want "cannot mutate a field"
