//go:build integration

package stable

import "github.com/lucasavila00/immutago/pkg/im"

type Reply struct {
	im.Local
	N int
}
type Score struct {
	im.Copied
	N int
}

func Read(value EntryProjection) Reply { return Reply{N: value.N()} }
