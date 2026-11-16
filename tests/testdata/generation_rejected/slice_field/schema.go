package slice_field

//go:generate go run github.com/lucasavila00/immutago/cmd/immutago-gen

import "github.com/lucasavila00/immutago/pkg/im"

type Table struct {
	im.Frozen
	Rows []int
}
