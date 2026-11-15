//go:generate go run github.com/lucasavila00/immutago/cmd/immutago-gen

package invalid_unsafe

import (
	"github.com/lucasavila00/immutago/pkg/im"
	"unsafe"
)

type Good struct {
	im.Frozen
	X int
}

func Number() int { return int(unsafe.Sizeof(im.StringProjection{})) }
