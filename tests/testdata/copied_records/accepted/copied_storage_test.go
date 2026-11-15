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
	read.Set(8)
	arrayRead.X = 9
	if read.X != 8 || arrayRead.X != 9 || projection.Point().X != 1 || projection.Points().At(0).X != 1 {
		t.Fatal("generated getter changed stored data")
	}
	scorer.point.Store(point)
	loaded := im.Copy(scorer.point.Load())
	loaded.Set(5)
	if loaded.X != 5 || im.Copy(scorer.point.Load()).X != 1 {
		t.Fatal("Copy changed the Cell")
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

func TestSavedPointersDoNotReachProtectedCopies(t *testing.T) {
	source := api.Point{X: 1}
	sourcePointer := &source.X
	stored := frozen.Snapshot{Point: source}
	local := Reply{Point: source}
	*sourcePointer = 2

	fieldCopy := stored.Point
	fieldPointer := &fieldCopy.X
	*fieldPointer = 3
	localCopy := local.Point
	localPointer := &localCopy.X
	*localPointer = 4

	scorer := &Scorer{}
	scorer.current.Store(stored)
	getterCopy := scorer.current.Load().Point()
	getterPointer := &getterCopy.X
	*getterPointer = 5

	scorer.point.Store(stored.Point)
	projectionCopy := im.Copy(scorer.point.Load())
	projectionPointer := &projectionCopy.X
	*projectionPointer = 6

	builder := []api.Point{stored.Point}
	sequence := im.FreezeSlice(builder)
	clone := sequence.Projection().Clone()
	clonePointer := &clone[0].X
	*clonePointer = 7

	if source.X != 2 || fieldCopy.X != 3 || localCopy.X != 4 || getterCopy.X != 5 ||
		projectionCopy.X != 6 || clone[0].X != 7 {
		t.Fatal("saved pointers did not change their own copies")
	}
	if stored.Point.X != 1 || local.Point.X != 1 || scorer.current.Load().Point().X != 1 ||
		im.Copy(scorer.point.Load()).X != 1 || sequence.At(0).X != 1 {
		t.Fatal("a saved pointer changed protected storage")
	}
}
