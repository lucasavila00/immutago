package plain

import (
	"context"
	"sync"

	"github.com/lucasavila00/immutago/pkg/supervise"
)

type Prices struct {
	supervise.Service[supervise.NoArgs]
	mu     sync.Mutex
	prices map[string]float64
}

func (*Prices) Run(ctx context.Context) { <-ctx.Done() }
func (p *Prices) Price(item string) (float64, bool) {
	p.mu.Lock()
	defer p.mu.Unlock()
	price, ok := p.prices[item]
	return price, ok
}
func (p *Prices) Read(item string) (error, float64, bool, error) {
	price, found := p.Price(item)
	return nil, price, found, nil
}
