package api

//go:generate go run github.com/lucasavila00/immutago/cmd/immutago-gen

import (
	"context"

	"github.com/lucasavila00/immutago/pkg/im"
	"github.com/lucasavila00/immutago/pkg/supervise"
)

type Count struct {
	im.Copied
	N int
}

type Frozen struct {
	im.Frozen
	N int
}

type View struct {
	im.Local
	Text  im.StringProjection
	Count Count
}

type Alias = View
type Failure = error
type Commands struct{}

func (Commands) Status() (Failure, bool, Failure) { return nil, true, nil }

type Pool struct {
	supervise.Service[supervise.NoArgs]
	Commands
}

func (*Pool) Run(context.Context)  {}
func (*Pool) Lookup() (View, bool) { return Pair() }
func (*Pool) Read() (Failure, View, [2]Count, bool, Failure) {
	return nil, View{Text: im.ProjectString("ready"), Count: Count{N: 7}}, [2]Count{{N: 7}, {N: 8}}, true, nil
}
func (p *Pool) Forward() (error, View, [2]Count, bool, error) { return p.Read() }
func (p *Pool) Named() (value Alias, found bool) {
	value, found = p.Lookup()
	return
}
func (*Pool) Zero() (value View, found bool, err error) { return }
func (*Pool) References() (supervise.Ref[Pool], bool, supervise.Ref[Pool]) {
	return supervise.Ref[Pool]{}, false, supervise.Ref[Pool]{}
}
func Pair() (View, bool)           { return View{Text: im.ProjectString("ready"), Count: Count{N: 7}}, true }
func Reverse() (bool, View)        { value, found := Pair(); return found, value }
func Variadic() (View, bool, bool) { value, found := Pair(); return value, found, false }
func Numbers() (int, bool)         { return 7, true }
func FrozenPair() (Frozen, View)   { return Frozen{N: 7}, View{} }

type StateError struct{ Values map[string]int }

func (e *StateError) Error() string { return "state error" }
func (*Pool) Errors() (error, bool, error) {
	state := &StateError{Values: map[string]int{"count": 1}}
	return state, false, state
}
