package service

import (
	"github.com/lucasavila00/immutago/pkg/im"
	"github.com/lucasavila00/immutago/pkg/supervise"
)

type ZeroRoute struct {
	im.Local
	Worker supervise.Ref[Counter]
}

func protectedZero(route ZeroRoute) bool { return route.Worker.IsZero() }

func protectedZeroMethod(route ZeroRoute) func() bool { return route.Worker.IsZero }

func capturedZero(route ZeroRoute) func() bool {
	return func() bool { return route.Worker.IsZero() }
}

func asyncZero(route ZeroRoute) {
	go route.Worker.IsZero()
	defer route.Worker.IsZero()
}

func guardedBorrow(ref supervise.Ref[Counter]) {
	if !ref.IsZero() {
		pointer := ref.Borrow()
		_ = pointer
	}
}
