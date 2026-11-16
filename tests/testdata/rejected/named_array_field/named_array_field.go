package rejected

import "github.com/lucasavila00/immutago/pkg/im"

type Count [2]int
type Value struct {
	im.Frozen
	N Count
}

//go:generate go run github.com/lucasavila00/immutago/cmd/immutago-gen
