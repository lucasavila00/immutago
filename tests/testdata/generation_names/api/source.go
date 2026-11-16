package v

//go:generate go run github.com/lucasavila00/immutago/cmd/immutago-gen

import "github.com/lucasavila00/immutago/pkg/im"

type Record struct {
	im.Frozen
	N int
}
type Alias = Record
type Pointer = *Record
