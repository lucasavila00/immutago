//go:generate go run github.com/lucasavila00/immutago/cmd/immutago-gen

package service

import "github.com/lucasavila00/immutago/pkg/supervise"

var invalidTarget supervise.Ref[int]

func replaceHandle(route ZeroRoute, next supervise.Ref[Counter]) { route.Worker = next }

func captureRoute(route ZeroRoute) func() *Counter {
	return func() *Counter { return route.Worker.Borrow() }
}
