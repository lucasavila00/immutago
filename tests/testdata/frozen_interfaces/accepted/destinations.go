package accepted

import (
	"example.com/consumer/api"
	"github.com/lucasavila00/immutago/pkg/im"
)

var foundResult bool

// ProjectionResults keeps only the Local result in a local binding.
func ProjectionResults(reader api.Reader) {
	local := api.NewReaderProjection(im.ProjectionOf(reader))
	var text im.StringProjection
	var state struct{ Found bool }
	text, state.Found = local.Read()
	var flags [1]bool
	text, flags[0] = local.Read()
	results := map[int]bool{}
	text, results[0] = local.Read()
	text, foundResult = local.Read()
	pointer := &state.Found
	text, *pointer = local.Read()
	_ = text
}
