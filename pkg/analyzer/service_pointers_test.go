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
var Current *Worker
type Holder struct{Value *Worker}
type HolderAlias=Holder
func MakeHolder()Holder{return Holder{}}
func MakeAnon()struct{Value *Worker}{return struct{Value *Worker}{}}
func AnonPair()(struct{Value *Worker},int){return MakeAnon(),1}
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
		{"construction", `func use(){value:=api.Worker{};q:=&value;q=new(api.Worker);q=&api.Worker{};_=q}`},
		{"promoted method", `func use(r api.Handle)int{q:=r.Borrow();return q.Promoted()}`},
		{"factory", `func use()int{q:=api.Pointer();return q.Count()}`},
		{"generic factory", `func identity[T any](value T)T{return value}
func use(r api.Handle)int{next:=identity(api.Pointer);q:=next();q=identity(r.Borrow)();return q.Count()}`},
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

// TestMixedServicePointerProducer keeps an im.Local result under the existing callback rules.
func TestMixedServicePointerProducer(t *testing.T) {
	api := sourcePass(t, servicePointerAPI+`
func Mixed()(*Worker,im.StringProjection){return Pointer(),im.StringProjection{}}`).Pkg
	api.SetImports(nil)
	if got := checkServicePointerConsumer(t, api, `func use(){next:=api.Mixed;_,_=next()}`); got == "" {
		t.Fatal("mixed Local producer contract accepted")
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
		{"global nil", `func use(){api.Current=nil}`},
		{"field nil", `func use(){value:=api.Holder{};value.Value=nil}`},
		{"holder nil", `func use(){_=api.Holder{Value:nil}}`},
		{"holder zero", `func use(){var value api.Holder;_=value}`},
		{"holder alias", `type Alias=api.HolderAlias`},
		{"holder factory", `func use(){value:=api.MakeHolder();_=value.Value}`},
		{"holder return", `func use()api.Holder{return api.MakeHolder()}`},
		{"nested holder", `type Nested struct{Values [1]api.Holder}`},
		{"anonymous holder", `func use(){value:=api.MakeAnon();_=value}`},
		{"anonymous tuple", `func use(){api.AnonPair()}`},
		{"field", `type Holder struct{Value *api.Worker}`},
		{"array", `func use(q *api.Worker){_=[1]*api.Worker{q}}`},
		{"box", `func use(q *api.Worker){var value any=q;_=value}`},
		{"capture", `func use(q *api.Worker){f:=func()int{return q.Count()};_=f}`},
		{"goroutine", `func use(q *api.Worker){go q.Count()}`},
		{"defer", `func use(q *api.Worker){defer q.Count()}`},
		{"generic", `func identity[T any](v T)T{return v};func use(q *api.Worker){_=identity(q)}`},
		{"bodyless input", `func retain(*api.Worker);func use(r api.Handle){retain(r.Borrow())}`},
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
		{"defined pointer", `type Pointer *api.Worker;func use(){var p Pointer;_=p}`},
		{"pointer address", `func use(q *api.Worker){_= &q}`},
		{"indirect pointer", `func use(q **api.Worker){_=q}`},
		{"assertion", `func use(value any){q:=value.(*api.Worker);_=q}`},
		{"imported generic im field", `var value api.Box[api.FrozenData]`},
	} {
		t.Run(test.name, func(t *testing.T) {
			if got := checkServicePointerConsumer(t, api, test.source); got == "" {
				t.Fatal("unsupported service pointer use accepted")
			}
		})
	}
}

// TestServiceReferenceConversions keeps the sealed Ref fields opaque after a defined conversion.
func TestServiceReferenceConversions(t *testing.T) {
	api := sourcePass(t, servicePointerAPI).Pkg
	api.SetImports(nil)
	if got := checkServicePointerConsumer(t, api, `
type Copy api.Handle
type Alias = Copy
type Holder struct{Value [1]Alias}
func identity[T any](value T)T{return value}
func use(ref api.Handle)api.Handle{
 copies:=[]Holder{{[1]Alias{Copy(ref)}}}
 makeCopy:=func()Alias{return identity(copies[0].Value[0])}
 var boxed any=identity(makeCopy)()
 restored:=api.Handle(boxed.(Alias))
 restored.Borrow().Count()
 return restored
}`); got != "" {
		t.Fatal(got)
	}
	if got := checkServicePointerConsumer(t, api, `
type Ordinary struct{Value *api.Worker}
type Copy Ordinary
var saved Copy`); got == "" {
		t.Fatal("ordinary pointer holder treated as a service reference")
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

// TestServiceValuesStayOrdinary keeps a service value distinct from its Local pointer.
func TestServiceValuesStayOrdinary(t *testing.T) {
	source := strings.Replace(servicePointerAPI, "Value int}", "Value int; Next *Worker}", 1)
	api := sourcePass(t, source).Pkg
	api.SetImports(nil)
	if got := checkServicePointerConsumer(t, api, `
type Holder struct{Value api.Worker}
func identity[T any](value T)T{return value}
func use(value api.Worker)any{
 other:=value
 change:=func(){other.Value++}
 change();go change();defer change()
 values:=[]api.Worker{identity(other)}
 return Holder{values[0]}
}`); got != "" {
		t.Fatal(got)
	}
}

// TestLocalPointerProducerContracts keeps im.Local pointer results outside producer contracts.
func TestLocalPointerProducerContracts(t *testing.T) {
	for _, source := range []string{
		`type Getter func() *LocalRecord`,
		`type Getter = func() *LocalRecord`,
		`type Reader interface{ Read() *LocalRecord }`,
		`type Reader = interface{ Read() *LocalRecord }`,
		`type Reader interface{im.FrozenInterface; Read() *LocalRecord}`,
		`type Getter func() (int, *LocalRecord)`,
		`func use(f func() *LocalRecord){p:=f();_=p}`,
	} {
		t.Run(source, func(t *testing.T) {
			if got := checkSource(t, `type LocalRecord struct{im.Local; N int};`+source); got == "" {
				t.Fatal("im.Local pointer producer contract accepted")
			}
		})
	}
}

// TestServiceValueFieldRules keeps field proofs and selected im targets checked.
func TestServiceValueFieldRules(t *testing.T) {
	for _, test := range []struct {
		name, source string
		reject       bool
	}{
		{"Frozen field", `type Bad struct{supervise.Service[int]; Data FrozenData}`, true},
		{"Local field", `type Data struct{im.Local; N int};type Bad struct{supervise.Service[int]; Data Data}`, true},
		{"protected target", `type WithData struct{supervise.Service[int]; Data *FrozenData}
func change(p WithData){p.Data.Value++}`, true},
		{"ordinary slot", `type WithData struct{supervise.Service[int]; Data *FrozenData}
func change(p WithData){p.Data=&FrozenData{Value:1}}`, false},
	} {
		t.Run(test.name, func(t *testing.T) {
			if got := checkSource(t, servicePointerAPI+test.source); (got != "") != test.reject {
				t.Fatalf("rejected=%t, diagnostics=%s", test.reject, got)
			}
		})
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
