package service

import (
	"github.com/lucasavila00/immutago/pkg/im"
	"github.com/lucasavila00/immutago/pkg/supervise"
)

type ZeroRoute struct {
	im.Local
	Worker supervise.Ref[Counter]
}

func protectedZeroMethod(route ZeroRoute) func() bool { return route.Worker.IsZero }

func capturedZero(route ZeroRoute) func() bool {
	return func() bool { return route.Worker.IsZero() }
}

func asyncZero(route ZeroRoute) {
	go route.Worker.IsZero()
	defer route.Worker.IsZero()
}

type EmbeddedZeroRoute struct {
	im.Local
	supervise.Ref[Counter]
}

func promotedZero(route EmbeddedZeroRoute) bool { return route.IsZero() }

type Status int

func (s Status) IsZero() bool { return s == 0 }

type MarkedStatus struct {
	im.Local
	Status
}

func userZero(status MarkedStatus) bool { return status.Status.IsZero() }

func addressedZero(route *ZeroRoute) bool { return (&route.Worker).IsZero() }
