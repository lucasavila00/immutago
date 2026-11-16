package local_value_field

//go:generate go run github.com/lucasavila00/immutago/cmd/immutago-gen

import (
	"github.com/lucasavila00/immutago/pkg/im"
)

type Local struct {
	im.Local
	Value int
}
type SavedValue struct {
	im.Frozen
	Value Local
}
