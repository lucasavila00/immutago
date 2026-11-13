package rejected

import "github.com/lucasavila00/immutago/pkg/im"

type Good struct {
	im.Marker
	Value int
	Array [2]int
}

var Bad = struct{ Value Good }{} // want "unmarked struct contains immutable field"
