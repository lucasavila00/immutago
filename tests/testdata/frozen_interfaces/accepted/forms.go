//go:generate go run github.com/lucasavila00/immutago/cmd/immutago-gen

package accepted

import (
	"example.com/consumer/api"
	"example.com/consumer/implementation"
	"github.com/lucasavila00/immutago/pkg/im"
	"github.com/lucasavila00/immutago/pkg/supervise"
)

type Model struct {
	im.Frozen
	Scorer  api.Scorer
	Readers [2]api.Reader
	List    im.FrozenSlice[api.Scorer]
	Map     im.FrozenMap[api.Scorer, api.Scorer]
}

type Reply struct {
	im.Local
	Scorer api.ScorerProjection
}

type Worker struct{ supervise.Service[api.Scorer] }

func (w *Worker) Scorer() api.ScorerProjection {
	return api.NewScorerProjection(im.ProjectionOf(w.Args()))
}

var current ModelCell
var slot im.Cell[api.Scorer]

func read(s api.Scorer) float64         { return s.Score([]float32{1}) }
func pair() (*implementation.Tree, int) { return &implementation.Tree{Weight: 3}, 1 }
func take(s api.Scorer, _ int) float64  { return read(s) }
func forwarded() (api.Scorer, int)      { return pair() }

func Exercise() float64 {
	var empty api.Scorer
	var nilValue api.Scorer = nil
	var pointer *implementation.Tree = nil
	var typedNil api.Scorer = pointer
	if empty != nil || nilValue != nil || typedNil == nil {
		return -1
	}
	value := api.Scorer(implementation.Count{N: 2})
	var reader api.Reader = implementation.Make()
	var aliased api.Alias = reader
	copy := aliased
	copy = reader
	copy = nil
	_ = copy
	_ = take(pair())
	values := []api.Scorer{value, reader}
	values = append(values, implementation.Count{N: 4})
	values[0] = &implementation.Tree{Weight: 5}
	list := im.FreezeSlice(values)
	entries := map[api.Scorer]api.Scorer{implementation.Count{N: 2}: reader}
	frozenMap := im.FreezeMap(entries)
	current.Store(Model{Scorer: reader, Readers: [2]api.Reader{reader}, List: list, Map: frozenMap})
	slot.Store(value)
	_ = slot.LoadShared().Score(nil)
	fromSlot := api.NewScorerProjection(slot.Load())
	_ = fromSlot.Score(nil)
	local := current.Load()
	local.Readers().At(0).Visit()
	name := local.Readers().At(0).Name()
	if !name.Is("tree") {
		return -2
	}
	fromList := api.NewScorerProjection(local.List().At(1))
	var shared api.Scorer
	shared, found := frozenMap.Get(value)
	shared, found = frozenMap.Get(value)
	_ = shared
	fromMap, found := local.Map().Get(value)
	fromMap, found = local.Map().Get(value)
	_, _ = local.Map().Get(value)
	if !found {
		return -3
	}
	mapReader := api.NewScorerProjection(fromMap)
	reply := Reply{Scorer: local.Scorer()}
	return reply.Scorer.Score([]float32{1}) + fromList.Score([]float32{1}) + mapReader.Score([]float32{1})
}

// ReadResults checks local bindings, direct forwarding, and interface projections.
func ReadResults(reader api.Reader, buf []byte) (float64, bool, []byte, error) {
	text, found := reader.Read()
	var other, ok = reader.Read()
	text, found = reader.Read()
	_, _ = reader.Read()
	_ = text
	_ = other
	_ = found
	_ = ok
	local := api.NewReaderProjection(im.ProjectionOf(reader))
	value, found, buf, err := local.Lookup([]byte{7}, buf)
	var score, present, output, failure = local.Lookup(nil, buf)
	value, found, buf, err = local.Lookup([]byte{7}, buf)
	_, _, _, _ = local.Lookup(nil, buf)
	_ = value
	_ = found
	_ = err
	_ = score
	_ = present
	_ = output
	_ = failure
	summary, yes, bytes, fail := local.SummaryLookup(api.Bytes(buf))
	_ = summary
	_ = yes
	_ = bytes
	_ = fail
	return forwardResults(local.Lookup([]byte{7}, buf))
}

func forwardResults(value float64, found bool, out []byte, err error) (float64, bool, []byte, error) {
	return value, found, out, err
}

// FrozenResults checks tuple copies and declared method results through local projections.
func FrozenResults(reader api.Reader) {
	pointer, count := pair()
	var another, number = pair()
	pointer, count = pair()
	_, _ = pair()
	_ = pointer
	_ = count
	_ = another
	_ = number
	local := api.NewReaderProjection(im.ProjectionOf(reader))
	persistent := local.Self()
	_ = persistent
	var tree implementation.TreeReader = &implementation.Tree{}
	treeLocal := implementation.NewTreeReaderProjection(im.ProjectionOf(tree))
	result := treeLocal.TreePointer()
	_ = result
}
