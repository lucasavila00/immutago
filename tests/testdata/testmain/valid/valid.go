//go:generate go run github.com/lucasavila00/immutago/cmd/immutago-gen

package valid

import "github.com/lucasavila00/immutago/pkg/im"

type Value struct {
	im.Frozen
	X int
}
