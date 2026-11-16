package accepted

import (
	"example.com/consumer/api"
	"github.com/lucasavila00/immutago/pkg/im"
)

var channel chan api.Scorer

type Holder struct{ Value api.Scorer }

func many(...api.Scorer) {}
func empty() api.Scorer  { return api.Scorer(nil) }

func mapDestinations(values im.FrozenMap[int, api.Scorer]) {
	var boxed any
	var found bool
	boxed, found = values.Get(0)
	_ = boxed
	_ = found
	var output [1]api.Scorer
	output[0], found = values.Get(0)
}
