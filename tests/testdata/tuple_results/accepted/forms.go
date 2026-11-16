package accepted

import (
	"example.com/consumer/api"
	"example.com/consumer/plain"
	"github.com/lucasavila00/immutago/pkg/im"
	"github.com/lucasavila00/immutago/pkg/supervise"
)

var Found bool
var SavedError error
var _, Initial = api.Pair()

func consume(value api.View, found bool)     { _ = value.Count.N; Found = found }
func variadic(value api.View, flags ...bool) { _ = value; Found = flags[0] }
func Forward() (api.Alias, bool)             { return (api.Pair)() }
func Named() (value api.Alias, found bool)   { value, found = api.Pair(); return }
func Zero() (value api.Alias, found bool)    { return }

func Bind(ref supervise.Ref[api.Pool], values []bool, destination *bool) {
	value, found := ref.Borrow().Lookup()
	consume(value, found)
	var next, ok = (api.Pair)()
	consume(next, ok)
	value, Found = ref.Borrow().Named()
	Found, value = api.Reverse()
	value, values[0] = api.Pair()
	value, *destination = api.Pair()
	var state struct{ OK bool }
	value, state.OK = ref.Borrow().Lookup()
	consume(value, state.OK)
	SavedError, value, _, Found, _ = ref.Borrow().Read()
	_ = value
	_, _, _, _, SavedError = ref.Borrow().Forward()
	_, _, _ = ref.Borrow().Status()
	consume(ref.Borrow().Lookup())
	variadic(api.Variadic())
	if view, exists := ref.Borrow().Lookup(); exists {
		consume(view, exists)
	}
	var foundAgain bool
	view, foundAgain := api.Pair()
	consume(view, foundAgain)
}

func Ordinary(callback func() (api.Count, bool), ref supervise.Ref[plain.Prices]) (any, bool) {
	count, found := callback()
	count.N++
	_, _, _, SavedError = ref.Borrow().Read("item")
	return count, found
}

func BuilderSlots() im.FrozenSlice[bool] {
	values := []bool{false}
	var view api.View
	view, values[0] = api.Pair()
	_ = view
	returnValue := im.FreezeSlice(values)
	return returnValue
}

func CopyPair(p im.SliceProjection[int]) ([]int, bool) { return p.Clone(), true }

func FrozenCallback(read func() (api.Frozen, bool)) (any, bool) { return read() }

func ProtectedBuilderSlots() im.FrozenSlice[api.Frozen] {
	values := []api.Frozen{{}}
	var view api.View
	values[0], view = api.FrozenPair()
	_ = view
	frozen := im.FreezeSlice(values)
	return frozen
}
