package rejected

import (
	"example.com/consumer/generic_callback_api"
	"github.com/lucasavila00/immutago/pkg/im"
)

type Good struct {
	im.Marker
	X int
}

var Global Good

func (Good) Ptr() *int { return &Global.X }

func bad() {
	generic_callback_api.Apply(Good{}) // want "borrow|shared reference|unsupported"
}
