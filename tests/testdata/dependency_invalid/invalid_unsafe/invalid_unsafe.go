package invalid_unsafe

import (
	"github.com/lucasavila00/immutago/pkg/im"
	"unsafe"
)

type Good struct {
	im.Snapshot
	X int
}

func Number() int { return int(unsafe.Sizeof(Good{})) }
