package mixedallowed

import "example.com/consumer/mixedapi"

// Read binds each Local result from one direct typed call.
func Read() int {
	worker, text := mixedapi.Mixed()
	return worker.Count() + text.Len()
}
