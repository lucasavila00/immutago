package rejected

import "github.com/lucasavila00/immutago/pkg/im"

type Count [2]int
type Value struct {
	im.Marker
	N Count
}
