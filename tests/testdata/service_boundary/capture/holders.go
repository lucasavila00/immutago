package capture

import (
	"example.com/consumer/api"
	"example.com/consumer/sealedapi"
)

type HolderAlias = api.HolderAlias
type NestedHolder struct{ Values [1]api.Holder }

var atomicCarrier sealedapi.Carrier

func holder() api.Holder { return api.MakeHolder() }

func readHolder() {
	var zero api.Holder
	_ = zero
	value := api.MakeHolder()
	worker := value.Value
	worker.Count()
	_ = api.Holder{Value: nil}
	anon := api.MakeAnon()
	_ = anon
	api.AnonPair()
}
