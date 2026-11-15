package service

import (
	"github.com/lucasavila00/immutago/pkg/im"
	"github.com/lucasavila00/immutago/pkg/supervise"
)

type Route struct {
	im.Local
	Worker supervise.Ref[Counter]
}

// boxRoute keeps the im restriction on interface storage.
func boxRoute(route Route) any { return route }

// convertRoute keeps the im restriction on explicit interface conversion.
func convertRoute(route Route) any { return any(route) }
