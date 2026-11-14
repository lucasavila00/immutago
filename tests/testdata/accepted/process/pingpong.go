package process

import (
	"context"

	"github.com/lucasavila00/immutago/pkg/proc"
)

// Ping is a process that calls a Pong.
type Ping struct {
	proc.Process[proc.PID[Pong]]
	answers int
}

// Run asks the Pong once and waits until ctx ends.
func (p *Ping) Run(ctx context.Context) {
	p.answers = p.Args().Proc().Answer()
	<-ctx.Done()
}

// Answer returns a constant answer.
func (p *Ping) Answer() int { return 1 }

// Pong is a process that calls a Ping.
type Pong struct {
	proc.Process[proc.PID[Ping]]
	answers int
}

// Run asks the Ping once and waits until ctx ends.
func (p *Pong) Run(ctx context.Context) {
	p.answers = p.Args().Proc().Answer()
	<-ctx.Done()
}

// Answer returns a constant answer.
func (p *Pong) Answer() int { return 2 }

// StartPingPong starts two processes that know each other's PID.
func StartPingPong(root *proc.Supervisor) proc.PID[Ping] {
	pong := proc.Reserve[Pong]()
	ping := proc.Start[Ping](root, pong)
	proc.StartReserved(root, pong, ping)
	return ping
}
