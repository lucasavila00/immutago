package service

import (
	"github.com/lucasavila00/immutago/pkg/im"
	"github.com/lucasavila00/immutago/pkg/supervise"
)

// Sample is immutable even when passed into a service.
type Sample struct {
	im.Scoped
	Rate float64
}

// Change attempts to write through a protected input.
func (*Refresher) Change(p *Sample) { p.Rate = 1 }

// Capture attempts to keep a Scoped pointer in a closure.
func (*Refresher) Capture(p *Sample) { f := func() float64 { return p.Rate }; _ = f }

// Accept takes an ordinary interface.
func (*Refresher) Accept(any) {}

// boxedInput attempts to hide a protected input in an interface.
func boxedInput(ref supervise.Ref[Refresher]) { ref.Current().Accept(&Sample{}) }

// Echo attempts to return an ordinary input through the immutable result boundary.
func (*Refresher) Echo(input any) any { return input }
