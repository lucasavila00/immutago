package rejected

import im "github.com/lucasavila00/immutago/pkg/immutable"

type Good struct {
	im.Marker
	Value int
	Array [2]int
}

type Bad struct{ Value map[string]Good } // want "unmarked struct contains immutable field"
