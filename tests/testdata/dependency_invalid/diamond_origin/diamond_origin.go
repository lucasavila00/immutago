package diamond_origin

import "github.com/lucasavila00/immutago/pkg/im"

type Broken struct {
	im.Frozen
	Values []int
}

//go:generate go run github.com/lucasavila00/immutago/cmd/immutago-gen
