package sealedapi

import (
	"context"
	"sync/atomic"

	"github.com/lucasavila00/immutago/pkg/supervise"
)

type Worker struct{ supervise.Service[int] }

func (*Worker) Run(context.Context) {}
func (*Worker) Count() int          { return 1 }

// Handle is the only public reference type; export data need not include Ref itself.
type Handle supervise.Ref[Worker]

// Carrier has a private field with the same type, but its field belongs to this package.
type Carrier struct{ value *atomic.Pointer[Worker] }
