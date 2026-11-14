package service

import (
	"context"

	"github.com/lucasavila00/immutago/pkg/im"
	"github.com/lucasavila00/immutago/pkg/supervise"
)

// Generator supplies one random value per decision.
type Generator interface{ Float64() float64 }

// Sequence is an ordinary mutable generator used by the consumer fixture.
type Sequence struct{ Next float64 }

// Float64 advances the sequence.
func (g *Sequence) Float64() float64 { g.Next += 0.1; return g.Next }

// SampleRequest sets the sample rate for one call.
type SampleRequest struct {
	im.Shared
	Rate float64
}

// Sampler accepts dependencies and mutable data on each call.
type Sampler struct {
	supervise.Service[supervise.NoArgs]
}

// Run waits for shutdown.
func (*Sampler) Run(ctx context.Context) { <-ctx.Done() }

// Decide reads immutable request data and calls an ordinary interface.
func (*Sampler) Decide(req SampleRequest, random Generator) bool {
	return random.Float64() < req.Rate
}

// Record changes caller-supplied storage and invokes an ordinary callback.
func (*Sampler) Record(counts map[string]int, samples []int, total *int, visit func(string), out chan string) {
	counts["api"]++
	samples[0]++
	(*total)++
	visit("api")
	out <- "api"
}

// Count accepts both individual values and expanded slices.
func (*Sampler) Count(values ...int) int { return len(values) }

// serviceInputs exercises input types through direct Current calls.
func serviceInputs(ref supervise.Ref[Sampler]) {
	ref.Current().Decide(SampleRequest{Rate: 0.5}, &Sequence{})
	count := 0
	ref.Current().Record(map[string]int{}, []int{0}, &count, func(string) {}, make(chan string, 1))
	ref.Current().Count(1, 2)
	values := []int{1, 2}
	ref.Current().Count(values...)
}
