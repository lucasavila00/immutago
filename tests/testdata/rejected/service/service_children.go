package service

import (
	"context"

	"github.com/lucasavila00/immutago/pkg/supervise"
)

// Fetcher is a service that a FanOut starts.
type Fetcher struct {
	supervise.Service[supervise.NoArgs]
}

// Run waits until ctx ends.
func (*Fetcher) Run(ctx context.Context) { <-ctx.Done() }

// FanOut uses the group of its run in each rejected form.
type FanOut struct {
	supervise.Service[supervise.NoArgs]
	sup *supervise.Group
}

var savedGroup *supervise.Group

// keep receives a group.
func keep(*supervise.Group) {}

// Run uses the group of the run in rejected forms.
func (f *FanOut) Run(ctx context.Context) {
	f.sup = f.Children()
	savedGroup = f.Children()
	keep(f.Children())
	f.Children().Stop()
	get := f.Children
	go supervise.Start[Fetcher](f.Children(), supervise.NoArgs{})
	func() { supervise.Start[Fetcher](f.Children(), supervise.NoArgs{}) }()
	sup := f.Children()
	keep(sup)
	other := f.Children()
	go func() { supervise.Start[Fetcher](other, supervise.NoArgs{}) }()
	var late = f.Children()
	supervise.StartGroup(late)
	_ = get
	<-ctx.Done()
}

// owned returns the group of the run.
func (f *FanOut) owned() *supervise.Group { return f.Children() }

// fetchUnder starts a fetcher under the group of another service.
func fetchUnder(ref supervise.Ref[FanOut]) {
	supervise.Start[Fetcher](ref.Current().Children(), supervise.NoArgs{})
}
