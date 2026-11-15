package service

import (
	"sync"

	"github.com/lucasavila00/immutago/pkg/im"
	"github.com/lucasavila00/immutago/pkg/supervise"
)

type Pool struct {
	supervise.Service[supervise.NoArgs]
	mu      sync.Mutex
	workers map[string]supervise.Ref[Counter]
}

func (p *Pool) Worker(name string) supervise.Ref[Counter] {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.workers[name]
}

type Route struct {
	im.Local
	Worker supervise.Ref[Counter]
}

func (p *Pool) Route(name string) Route { return Route{Worker: p.Worker(name)} }

func UseRoute(route Route) int { return route.Worker.Current().Count() }
