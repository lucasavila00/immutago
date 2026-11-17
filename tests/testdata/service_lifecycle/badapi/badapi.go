package badapi

import (
	"context"

	"github.com/lucasavila00/immutago/pkg/supervise"
)

type Generic[A any] struct{ supervise.Service[A] }

func (Generic[A]) Run(context.Context) {}

type Alias[A any] = Generic[A]
type Defined[A any] Generic[A]

func (Defined[A]) Run(context.Context) {}

func Start[A any](root *supervise.Supervisor, args A) (supervise.Ref[Alias[A]], error) {
	return supervise.Start[Alias[A]](root, args)
}
