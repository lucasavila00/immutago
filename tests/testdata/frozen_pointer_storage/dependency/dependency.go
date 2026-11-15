package dependency

import "example.com/consumer/api"

// Only an ordinary carrier appears in this package's input types.
func Change(state *api.State) {
	state.Settings.Limit = 1
}

func Promoted(outer *api.Outer) {
	outer.Limit++
}
