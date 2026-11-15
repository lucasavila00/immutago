package rejected

import (
	"example.com/consumer/api"
	"example.com/consumer/implementation"
	"github.com/lucasavila00/immutago/pkg/im"
	"github.com/lucasavila00/immutago/pkg/supervise"
)

var saved api.ScorerProjection
var channel chan api.Scorer

type Holder struct{ Value api.Scorer }
type Worker struct{ supervise.Service[api.Scorer] }

func (w *Worker) Raw() api.Scorer { return w.Args() }

func forms(s api.Scorer, local api.ScorerProjection) {
	var ordinary any = s
	_ = ordinary
	_ = s.(*implementation.Tree)
	_ = api.Scorer.Score(s, nil)
	method := s.Score
	_ = method
	go s.Score(nil)
	defer s.Score(nil)
	_ = func() float64 { return s.Score(nil) }
	_ = func() float64 { return local.Score(nil) }
	_ = api.Scorer(nil)
	var implicit api.Scorer = &implementation.Count{}
	_ = implicit
	saved = local
	_ = im.Copy(im.ProjectionOf(s))
	_ = identity(s)
	switch s.(type) {
	case *implementation.Tree:
	}
}

func identity[T any](v T) T                          { return v }
func raw(local im.Projection[api.Scorer]) api.Scorer { return local.Read() }
func rawPointer(local im.Projection[api.Scorer]) *implementation.Tree {
	return local.Read().(*implementation.Tree)
}
func many(...api.Scorer)    {}
func boxed(...any)          {}
func variadic(s api.Scorer) { boxed(s) }

func mutation(tree *implementation.Tree, scorer api.Scorer) {
	tree.Weight = 4
	values := []api.Scorer{scorer}
	frozen := im.FreezeSlice(values)
	_ = frozen
	_ = values[0]
}

func localStorage(local api.ScorerProjection) {
	_ = im.Cell[api.ScorerProjection]{}
	values := []api.ScorerProjection{local}
	_ = im.FreezeSlice(values)
}
