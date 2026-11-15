package diamond_origin

import "github.com/lucasavila00/immutago/pkg/im"

type Broken struct {
	im.Snapshot
	Values []int
}
