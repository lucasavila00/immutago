package analyzer

import (
	"go/types"
	"strings"
	"testing"

	"golang.org/x/tools/go/analysis"
)

const servicePointerAPI = `import (
"context"
"github.com/lucasavila00/immutago/pkg/supervise"
)
var _ im.Frozen
type View struct{}
func (View) Promoted() int {return 1}
type Worker struct {supervise.Service[int]; View; Value int}
func (*Worker) Run(context.Context) {}
func (p *Worker) Count() int {return p.Value}
type Alias = Worker
type Handle = supervise.Ref[Worker]
func Pointer() *Worker {return new(Worker)}
func Pair() (*Worker,int) {return Pointer(),1}
type Anonymous = struct {supervise.Service[int]; View}
type AnonymousHandle = supervise.Ref[Anonymous]
type Box[T any] struct {Value T}
type FrozenData struct {im.Frozen; Value int}
`

// TestLocalServicePointers accepts local use through concrete and indirect producers.
func TestLocalServicePointers(t *testing.T) {
	api := sourcePass(t, servicePointerAPI).Pkg
	api.SetImports(nil)
	for _, test := range []struct{ name, source string }{
		{"binding", `func use(r api.Handle) int {q:=r.Borrow();other:=q;q=r.Borrow();_=other;return q.Count()}`},
		{"nil", `func use(){var q *api.Worker;q=nil;_=q}`},
		{"comparison", `func use(r api.Handle) bool {q:=r.Borrow();return q==nil || q==r.Borrow()}`},
		{"parameter", `func read(q *api.Worker)int{return q.Count()}
func use(r api.Handle)int{q:=r.Borrow();return read(q)+read(r.Borrow())}`},
		{"alias", `type Pointer = *api.Alias;func read(q Pointer)int{return q.Count()}`},
		{"promoted method", `func use(r api.Handle)int{q:=r.Borrow();return q.Promoted()}`},
		{"factory", `func use()int{q:=api.Pointer();return q.Count()}`},
		{"tuple", `func use()int{q,n:=api.Pair();return q.Count()+n}`},
		{"interface", `type Borrower interface{Borrow()*api.Worker}
func use(r api.Handle)int{var b Borrower=r;q:=b.Borrow();return q.Count()}`},
		{"accessor", `func use(r api.Handle)int{next:=r.Borrow;q:=next();return q.Count()}`},
		{"generic producer", `func read[T interface{Borrow()*api.Worker}](r T)int{q:=r.Borrow();return q.Count()}
func use(r api.Handle)int{return read(r)}`},
		{"anonymous", `func use(r api.AnonymousHandle)int{q:=r.Borrow();return q.Promoted()}`},
	} {
		t.Run(test.name, func(t *testing.T) {
			if got := checkServicePointerConsumer(t, api, test.source); got != "" {
				t.Fatal(got)
			}
		})
	}
}

// TestLocalServicePointerBoundary rejects retention and target access by resolved pointer type.
func TestLocalServicePointerBoundary(t *testing.T) {
	api := sourcePass(t, servicePointerAPI).Pkg
	api.SetImports(nil)
	for _, test := range []struct{ name, source string }{
		{"return", `func use(q *api.Worker)*api.Worker{return q}`},
		{"nil return", `func use()*api.Worker{return nil}`},
		{"global", `var saved *api.Worker`},
		{"field", `type Holder struct{Value *api.Worker}`},
		{"array", `func use(q *api.Worker){_=[1]*api.Worker{q}}`},
		{"box", `func use(q *api.Worker){var value any=q;_=value}`},
		{"capture", `func use(q *api.Worker){f:=func()int{return q.Count()};_=f}`},
		{"goroutine", `func use(q *api.Worker){go q.Count()}`},
		{"defer", `func use(q *api.Worker){defer q.Count()}`},
		{"generic", `func identity[T any](v T)T{return v};func use(q *api.Worker){_=identity(q)}`},
		{"read field", `func use(q *api.Worker)int{return q.Value}`},
		{"write field", `func use(q *api.Worker){q.Value++}`},
		{"dereference", `func use(q *api.Worker){_=*q}`},
		{"method value", `func use(q *api.Worker){f:=q.Count;_=f}`},
		{"Init", `func use(q *api.Worker){_=q.Init()}`},
		{"Run", `func use(q *api.Worker){q.Run(nil)}`},
		{"Args", `func use(q *api.Worker){_=q.Args()}`},
		{"Children", `func use(q *api.Worker){_=q.Children()}`},
		{"indirect return", `func use(b interface{Borrow()*api.Worker})*api.Worker{return b.Borrow()}`},
		{"anonymous return", `func use(q *api.Anonymous)*api.Anonymous{return q}`},
		{"imported generic im field", `var value api.Box[api.FrozenData]`},
	} {
		t.Run(test.name, func(t *testing.T) {
			if got := checkServicePointerConsumer(t, api, test.source); got == "" {
				t.Fatal("unsupported service pointer use accepted")
			}
		})
	}
}

// TestServiceReferenceOwnerFields keeps valid owner-package fields hidden by an opaque Ref.
func TestServiceReferenceOwnerFields(t *testing.T) {
	source := strings.Replace(servicePointerAPI, "Value int}", "Value int; Next *Worker}", 1)
	api := sourcePass(t, source).Pkg
	api.SetImports(nil)
	if got := checkServicePointerConsumer(t, api, `var saved api.Handle
func use(r api.Handle)any{saved=r;return r}`); got != "" {
		t.Fatal(got)
	}
}

// checkServicePointerConsumer checks a consumer with no direct im or supervise import.
func checkServicePointerConsumer(t *testing.T, api *types.Package, source string) string {
	t.Helper()
	pass := boundaryPass(t, dependencyImporter{"sample": api}, `package consumer;import api "sample";`+source)
	var reports []string
	pass.Report = func(d analysis.Diagnostic) { reports = append(reports, d.Message) }
	if _, err := run(pass); err != nil {
		t.Fatal(err)
	}
	return strings.Join(reports, "\n")
}
