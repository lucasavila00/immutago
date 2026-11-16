package unsafe_field

//go:generate go run github.com/lucasavila00/immutago/cmd/immutago-gen

import (
	"github.com/lucasavila00/immutago/pkg/im"
	u "unsafe"
)

type UnsafeField struct {
	im.Frozen
	Pointer u.Pointer
}
