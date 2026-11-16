package embedded_pointer

//go:generate go run github.com/lucasavila00/immutago/cmd/immutago-gen

import (
	"github.com/lucasavila00/immutago/pkg/im"
)

type Count int
type Pointer struct {
	im.Frozen
	*Count
}
