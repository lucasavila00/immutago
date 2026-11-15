package valid

import (
	"sync"

	"example.com/consumer/api"
	"github.com/lucasavila00/immutago/pkg/im"
)

var requests = sync.Pool{New: func() any { return &api.State{} }}

func Begin(cell *api.SettingsCell) *api.State {
	state := requests.Get().(*api.State)
	state.Begin(cell)
	return state
}

func End(state *api.State) {
	state.Reset()
	requests.Put(state)
}

func Containers(p *api.Settings) int {
	values := []*api.Settings{p}
	other := values
	other[0] = nil
	values = append(values, p)
	values = append(values, other...)
	copy(other, values[1:])
	versions := map[string]*api.Settings{"current": p}
	aliases := versions
	aliases["current"] = nil
	versions["next"] = p
	selected, found := versions["next"]
	selected, found = aliases["next"]
	var next, present = versions["next"]
	_ = found
	_ = present
	keys := map[*api.Settings]int{p: 3}
	count, ok := keys[p]
	_ = ok
	array := [2]*api.Settings{p, nil}
	pointer := &array
	pointer[1] = selected
	delete(aliases, "current")
	clear(other)
	return next.Read() + pointer[1].Limit + count
}

func Carriers(state *api.State) any {
	copy := *state
	copy.Settings = nil
	change := state.Reset
	change()
	callback := func() { state.Calls++ }
	callback()
	return identity(state)
}

func identity[T any](value T) T { return value }

func Promoted(p *api.Settings) int {
	var outer api.Outer
	outer.Settings = p
	value := outer.Limit
	outer.Settings = nil
	return value
}

func FreezePointers(p *api.Settings) im.FrozenSlice[*api.Settings] {
	values := []*api.Settings{p}
	frozen := im.FreezeSlice(values)
	return frozen
}

type Selected *api.Settings

func Handles(p *api.Settings) int {
	var value any = p
	selected := value.(*api.Settings)
	selected = identity(selected)
	slot := &selected
	*slot = p
	callback := selected.Read
	count := callback()
	channel := make(chan *api.Settings, 1)
	channel <- selected
	read, ok := <-channel
	_ = ok
	count += read.Value()
	count += (*api.Settings).Read(selected)
	count += Selected(selected).Limit
	return count
}

func Capture(p *api.Settings) func() int {
	return func() int { return p.Limit }
}

// ChangeCopy changes a named scalar copy through an ordinary interface.
func ChangeCopy(p *api.Settings) api.Count {
	count := p.Count
	var value interface{ Inc() } = &count
	value.Inc()
	return count
}
