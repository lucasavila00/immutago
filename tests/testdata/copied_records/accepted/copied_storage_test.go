package accepted

import (
	"example.com/consumer/api"
	"example.com/consumer/frozen"
	"github.com/lucasavila00/immutago/pkg/im"
	"testing"
)

func TestIndependentValues(t *testing.T) {
	scorer := &Scorer{}
	scorer.state.Result = api.NewScore(3)
	first := scorer.Last()
	scorer.state.Result = api.NewScore(4)
	if first.Value() != 3 || scorer.Last().Value() != 4 {
		t.Fatal("service result changed")
	}
	point := api.Point{X: 1, Y: 2}
	scorer.current.Store(frozen.Snapshot{Point: point, Points: [2]api.Point{point}, Score: first})
	projection := scorer.current.Load()
	read := projection.Point()
	arrayRead := projection.Points().At(0)
	read = api.Point{X: 8}
	arrayRead = api.Point{X: 9}
	if read.X != 8 || arrayRead.X != 9 || projection.Point().X != 1 || projection.Points().At(0).X != 1 {
		t.Fatal("generated getter changed stored data")
	}
	scorer.point.Store(point)
	loaded := im.Value(scorer.point.Load())
	loaded = api.Point{X: 5}
	if loaded.X != 5 || im.Value(scorer.point.Load()).X != 1 {
		t.Fatal("Value changed the Cell")
	}
	copyFromCollections(point)
	if NativeStorage(point) != 7 || LocalDeclaration() != 2 {
		t.Fatal("ordinary copied storage failed")
	}
}

func copyFromCollections(point api.Point) {
	first := collectionFirst(point)
	if first.X != point.X {
		panic("collection changed")
	}
}

func collectionFirst(point api.Point) api.Point {
	return firstResult(CopiedCollections(point))
}

func firstResult(first, second api.Point) api.Point {
	if first != second {
		panic("clone changed a frozen element")
	}
	return first
}
