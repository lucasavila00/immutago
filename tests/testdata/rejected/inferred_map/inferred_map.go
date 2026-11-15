package rejected

import "github.com/lucasavila00/immutago/pkg/im"

type Good struct {
	im.Snapshot
	Value int
	Array [2]int
}

var Bad = im.FreezeMap(map[string][]int{"a": {1}})
