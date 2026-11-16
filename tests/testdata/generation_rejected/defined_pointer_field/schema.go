package defined_pointer_field

//go:generate go run github.com/lucasavila00/immutago/cmd/immutago-gen

import "github.com/lucasavila00/immutago/pkg/im"

type Selected *Settings
type Bad struct {
	im.Frozen
	Value Selected
}
