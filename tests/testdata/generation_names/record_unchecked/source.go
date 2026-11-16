package record_unchecked

//go:generate go run github.com/lucasavila00/immutago/cmd/immutago-gen

import "github.com/lucasavila00/immutago/pkg/im"

type Record struct {
	im.Frozen
	Unchecked int
}
