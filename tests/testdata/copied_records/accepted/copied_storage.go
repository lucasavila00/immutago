package accepted

import (
	"example.com/consumer/api"
	"example.com/consumer/frozen"
	"github.com/lucasavila00/immutago/pkg/im"
	"github.com/lucasavila00/immutago/pkg/supervise"
)

type State struct {
	Result   api.Score
	Point    api.Point
	Attempts int
	Points   []api.Point
	ByPoint  map[api.Point]api.Point
}

type Scorer struct {
	supervise.Service[supervise.NoArgs]
	state   State
	current frozen.SnapshotCell
	point   im.Cell[api.Point]
}

type Reply struct {
	im.Local
	Score api.Score
	Point api.Point
}

func (s *Scorer) Last() api.Score                     { return s.state.Result }
func (s *Scorer) Point() api.Point                    { return s.state.Point }
func (s *Scorer) Points() [2]api.Point                { return [2]api.Point{s.state.Point} }
func (s *Scorer) Reply() Reply                        { return Reply{Score: s.state.Result, Point: s.state.Point} }
func (s *Scorer) Snapshot() frozen.SnapshotProjection { return s.current.Load() }

func LastReference(reference supervise.Ref[Scorer]) api.Score {
	return reference.Current().Last()
}

func Request(scorer *Scorer) State {
	state := State{Result: scorer.Last()}
	state.Attempts++
	state.Result = api.NewScore(3)
	return state
}

func (s *State) Replace(value api.Point) {
	s.Point = value
	s.Attempts++
}

var saved api.Summary

func Store(value api.Summary) State {
	saved = value
	state := State{Result: value.Score}
	state.Point = value.Points[0][0]
	state.Attempts++
	return state
}

func NativeStorage(value api.Point) int {
	values := []api.Point{value}
	alias := values
	alias[0] = api.Point{X: 2}
	values = append(values, value)
	part := values[1:]
	array := [2]api.Point{value}
	array[1] = value
	destination := make([]api.Point, 2)
	copy(destination, values)
	byPoint := map[api.Point]api.Point{value: value}
	byPoint[api.Point{}] = value
	delete(byPoint, api.Point{})
	clear(byPoint)
	state := &State{Point: value}
	state.Replace(array[1])
	return part[0].Sum() + api.Values(destination...) + state.Attempts
}

func LocalDeclaration() int {
	_ = api.Named{X: 1}
	type Count struct {
		im.Copied
		N int
	}
	value := Count{N: 1}
	value = Count{N: value.N + 1}
	other := struct {
		im.Copied
		N int
	}{N: value.N}
	return other.N
}

func CopiedCollections(value api.Point) (api.Point, api.Point) {
	values := []api.Point{value}
	sequence := im.FreezeSlice(values)
	read := im.Copy(sequence.Projection().At(0))
	clone := sequence.Projection().Clone()
	alias := clone
	alias[0] = api.Point{X: 9}
	entries := im.FreezeMap(map[api.Point]api.Point{value: value})
	mapping := entries.Projection().Clone()
	mapping[value] = api.Point{X: 8}
	return read, sequence.At(0)
}
