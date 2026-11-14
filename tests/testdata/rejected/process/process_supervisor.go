package process

import (
	"context"

	"github.com/lucasavila00/immutago/pkg/proc"
)

// Fetcher is a process that a FanOut starts.
type Fetcher struct{ proc.Process[proc.None] }

// Run waits until ctx ends.
func (*Fetcher) Run(ctx context.Context) { <-ctx.Done() }

// FanOut uses the supervisor of its run in each rejected form.
type FanOut struct {
	proc.Process[proc.None]
	sup *proc.Supervisor
}

var savedSupervisor *proc.Supervisor

// keep receives a supervisor.
func keep(*proc.Supervisor) {}

// Run uses the supervisor of the run in rejected forms.
func (f *FanOut) Run(ctx context.Context) {
	f.sup = f.Supervisor()
	savedSupervisor = f.Supervisor()
	keep(f.Supervisor())
	f.Supervisor().Stop()
	get := f.Supervisor
	go proc.Start[Fetcher](f.Supervisor(), proc.None{})
	func() { proc.Start[Fetcher](f.Supervisor(), proc.None{}) }()
	sup := f.Supervisor()
	keep(sup)
	other := f.Supervisor()
	go func() { proc.Start[Fetcher](other, proc.None{}) }()
	var late = f.Supervisor()
	proc.StartSupervisor(late)
	_ = get
	<-ctx.Done()
}

// owned returns the supervisor of the run.
func (f *FanOut) owned() *proc.Supervisor { return f.Supervisor() }

// fetchUnder starts a fetcher under the supervisor of another process.
func fetchUnder(pid proc.PID[FanOut]) { proc.Start[Fetcher](pid.Proc().Supervisor(), proc.None{}) }
