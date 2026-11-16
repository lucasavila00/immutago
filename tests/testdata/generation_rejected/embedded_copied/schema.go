package embedded_copied

//go:generate go run github.com/lucasavila00/immutago/cmd/immutago-gen

import (
	"github.com/lucasavila00/immutago/pkg/im"
)

type Stats struct {
	im.Copied
	N int
}
type Record struct {
	im.Frozen
	Stats
}
