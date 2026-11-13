package rejected

import "github.com/lucasavila00/immutago/pkg/im"

type Good struct {
	im.Marker
	X int
}

func bad(values map[string]any) { pointer := values["key"].(*Good); pointer.X = 2 } // want "borrow|shared reference|unsupported"
