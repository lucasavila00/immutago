//go:generate go run github.com/lucasavila00/immutago/cmd/immutago-gen

package privateapi

import "github.com/lucasavila00/immutago/pkg/im"

type privateReader interface {
	im.FrozenInterface
	Read() int
}
type Public = privateReader
