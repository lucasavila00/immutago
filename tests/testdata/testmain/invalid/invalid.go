package invalid

import "github.com/lucasavila00/immutago/pkg/im"

type Value struct {
	im.Snapshot
	X int
}
