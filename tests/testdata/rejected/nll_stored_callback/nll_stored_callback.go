package rejected

import (
	"example.com/consumer/callback_api"
	"github.com/lucasavila00/immutago/pkg/im"
)

type Good struct {
	im.Marker
	X int
}

var Global Good

func field() *int { return &Global.X }

func bad() { callback_api.Store(field); callback_api.ApplyStored() } // want "borrow|shared reference|unsupported"
