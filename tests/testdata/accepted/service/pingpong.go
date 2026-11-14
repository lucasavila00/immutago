package service

import (
	"context"

	"github.com/lucasavila00/immutago/pkg/supervise"
)

// Ping is a service that calls a Pong.
type Ping struct {
	supervise.Service[supervise.Ref[Pong]]
	answers int
}

// Run asks the Pong once and waits until ctx ends.
func (p *Ping) Run(ctx context.Context) {
	p.answers = p.Args().Current().Answer()
	<-ctx.Done()
}

// Answer returns a constant answer.
func (p *Ping) Answer() int { return 1 }

// Pong is a service that calls a Ping.
type Pong struct {
	supervise.Service[supervise.Ref[Ping]]
	answers int
}

// Run asks the Ping once and waits until ctx ends.
func (p *Pong) Run(ctx context.Context) {
	p.answers = p.Args().Current().Answer()
	<-ctx.Done()
}

// Answer returns a constant answer.
func (p *Pong) Answer() int { return 2 }

// StartPingPong starts two services that know each other's Ref.
func StartPingPong(root *supervise.Group) supervise.Ref[Ping] {
	pong := supervise.Reserve[Pong]()
	ping, err := supervise.Start[Ping](root, pong)
	if err != nil {
		panic(err)
	}
	if err := supervise.StartReserved(root, pong, ping); err != nil {
		panic(err)
	}
	return ping
}
