package rejected

import "github.com/lucasavila00/immutago/pkg/im"

type Good struct {
	im.Marker
	X int
}

func bad(payload any) { pointer := payload.(*Good); pointer.X = 2 } // want "borrow|shared reference|unsupported"
