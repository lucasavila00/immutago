package rejected

import im "github.com/lucasavila00/immutago/pkg/im"

type Count [2]int
type Value struct { // want "named array|embed im.Marker"
	im.Marker
	N Count
}
