package analyzer

import "testing"

const handleDefinitions = serviceDefinitions + `
func(c *Counter) Count() int {return c.count}
type Handle = supervise.Ref[Counter]
type Route struct { im.Local; Worker Handle }
`

// TestServiceHandles checks reference storage, interface use, and direct service calls.
func TestServiceHandles(t *testing.T) {
	cases := []struct{ name, source string }{
		{"message", `func use(r Handle) int{return r.Borrow().Count()}`},
		{"parentheses", `func use(r Handle) int{return ((r.Borrow)()).Count()}`},
		{"ref result", `func(*Counter) Worker() Handle {return Handle{}}`},
		{"marked result", `func(*Counter) Route() Route {return Route{}}`},
		{"marked field", `func use(r Route) int{return r.Worker.Borrow().Count()}`},
		{"anonymous target", `type Reader struct{};func(Reader) Count()int{return 1}
func use(r supervise.Ref[struct{supervise.Service[supervise.NoArgs];Reader}])int{return r.Borrow().Count()}`},
		{"shared field", `type SnapshotRoute struct{im.Frozen;Worker Handle}
func use(r *SnapshotRoute) int{return r.Worker.Borrow().Count()}`},
		{"native map", `func use(r Handle) int{m:=map[string]Handle{"a":r};v,ok:=m["a"];if !ok{return 0}
return v.Borrow().Count()}`},
		{"multiple copy", `func use(r Handle){a,b:=r,r;a,b=b,a;_,_=a,b}`},
		{"closure", `func use(r Handle){f:=func(){r.Borrow().Count()};go f();defer r.Borrow().Count()}`},
		{"discarded message result", `func(*Counter) Worker()Handle{return Handle{}}
func use(r Handle){go r.Borrow().Worker();defer r.Borrow().Worker()}`},
		{"channel", `func use(r Handle,ch chan Handle){ch<-r;other:=<-ch;other.Borrow().Count()}`},
		{"concrete callback", `func use(r Handle,f func(Handle)){f(r)}`},
		{"concrete generic parameter", `func use[T any](r Handle,v T)Handle{return r}
func call(r Handle){use(r,1).Borrow().Count()}`},
		{"concrete generic field", `type Box[T any]struct{Worker Handle;Value T}
func use(r Handle){box:=Box[int]{Worker:r};box.Worker.Borrow().Count()}`},
		{"concrete generic alias", `type Alias[T any]=Handle;func use(r Alias[int]){r.Borrow().Count()}`},
		{"unrelated bodyless function", `func cpuID()uint64`},
		{"callback result", `func use(f func()Handle)int{next:=f;return next().Borrow().Count()}`},
		{"concrete iterator", `func use(r Handle){
for next:=range func(yield func(Handle)bool){yield(r)}{next.Borrow().Count()}}`},
		{"producer interface", `type Producer interface{Worker()Handle}
func pass(p Producer)Producer{return p}
func use(p Producer)int{next:=p;return pass(next).Worker().Borrow().Count()}`},
		{"native operations", `func use(r Handle){rs:=make([]Handle,1);rs=append(rs,r);copy(rs,rs);clear(rs)
m:=make(map[Handle]int);m[r]=1;delete(m,r);_=len(rs);_=cap(rs)}`},
		{"service reference state", `type Pool struct{supervise.Service[Handle];Workers map[string]Handle}
func(p *Pool) Update(r Handle){p.Workers["x"]=r;self:=p;f:=func(){self.Workers["y"]=r};f()
p.Args().Borrow().Count()}`},
		{"service callback", `func(c *Counter) Visit(f func(*Counter)){f(c)}
func use(r Handle){r.Borrow().Visit(func(c *Counter){c.count++})}`},
		{"frozen field", `type Routes struct{im.Frozen;Workers im.FrozenSlice[Handle]}
func use(r Routes)int{return r.Workers.At(0).Borrow().Count()}`},
		{"interface", `func use(r Handle){var v interface{Borrow()*Counter}=r;_=v.Borrow()}`},
		{"any", `func use(r Handle){var v any=r;_=v}`},
		{"call", `func consume(any){};func use(r Handle){consume(r)}`},
		{"tuple call", `func pair(r Handle)(Handle,int){return r,1};func consume(any,int){}
func use(r Handle){consume(pair(r))}`},
		{"interface return", `func use(r Handle)any{return r}`},
		{"interface literal", `func use(r Handle){_=struct{Value any}{r}}`},
		{"interface channel", `func use(r Handle,ch chan any){ch<-r}`},
		{"interface comparison", `func use(r Handle,v any)bool{return r==v}`},
		{"interface range", `func use(rs []Handle){var v any;for _,v=range rs{_=v}}`},
		{"nested interface", `type Wrapper struct{Value Handle};func use(r Handle){var v any=Wrapper{r};_=v}`},
		{"assertion", `func use(v any)Handle{return v.(Handle)}`},
		{"callback boxing", `func use(r Handle){var v any=func()Handle{return r};_=v}`},
		{"callback assertion", `func use(v any){_=v.(func()Handle)}`},
		{"iterator boxing", `func use(r Handle){var v any
for v=range func(yield func(Handle)bool){yield(r)}{_=v}}`},
		{"producer boxing", `type Producer interface{Worker()Handle};func use(p Producer){var v any=p;_=v}`},
		{"producer assertion", `type Producer interface{Worker()Handle};func use(v any){_=v.(Producer)}`},
		{"type switch", `func use(v any){switch r:=v.(type){case Handle:r.Borrow().Count()}}`},
		{"explicit interface conversion", `func use(r Handle){_=any(r);_=interface{Borrow()*Counter}(r)}`},
		{"embedded interface conversion", `type Wrapper struct{Handle}
func use(r Handle){v:=interface{Borrow()*Counter}(Wrapper{r});_=v.Borrow()}`},
		{"conversion", `type Copy Handle;func use(r Handle){_=Copy(r)}`},
		{"conversion result", `type Copy Handle;func use(){_=Handle(Copy{})}`},
		{"nil conversion", `func use(){_=(*Handle)(nil)}`},
		{"nil container conversion", `func use(){_=[]Handle(nil)}`},
		{"callback generic", `func keep[T any](v T)T{return v}
func use(r Handle){_=keep(func()Handle{return r})}`},
		{"generic", `func get[T interface{Borrow()*Counter}](r T)*Counter{return r.Borrow()}
func use(r Handle){_=get(r)}`},
		{"bodyless parameter", `func leak(Handle)any`},
		{"bodyless opaque target", `type Worker struct{supervise.Service[supervise.NoArgs]; data im.Cell[Data]}
type Data struct{im.Frozen; N int};func(*Worker) Run(context.Context){};func read(supervise.Ref[Worker])`},
		{"bodyless result", `func leak()Handle`},
		{"bodyless callback", `func leak(func()Handle)any`},
		{"bodyless receiver", `type Wrapper struct{Worker Handle};func(Wrapper) leak()any`},
		{"generic wrapper", `type Box[T any] struct{Value T};func use(r Handle){_=Box[Handle]{r}}`},
		{"generic alias", `type Alias[T any]=Handle;var ref Alias[Handle]`},
		{"nil holder interface", `type Server struct{cache Handle}
func(*Server) OnFailure(error){};var _ interface{OnFailure(error)}=(*Server)(nil)`},
		{"named pointer conversion", `type Server struct{cache Handle};type Pointer *Server
func use(p *Server){_=Pointer(p);_=(*Server)(Pointer(p))}`},
		{"implementation", `func(c *Counter) Add(){self:=c;f:=func(){self.count++};f()}`},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := checkSource(t, handleDefinitions+tc.source); got != "" {
				t.Fatal(got)
			}
		})
	}
}

// TestServiceOwnerPointers keeps pointers ordinary inside the service's declaring package.
func TestServiceOwnerPointers(t *testing.T) {
	cases := []struct{ name, source string }{
		{"recovered pointer", `func use(v any)*Counter{return v.(Handle).Borrow()}`},
		{"pointer binding", `func use(r Handle){p:=r.Borrow();_=p}`},
		{"pointer return", `func use(r Handle)*Counter{return r.Borrow()}`},
		{"field read", `func use(r Handle)int{return r.Borrow().count}`},
		{"field write", `func use(r Handle){r.Borrow().count++}`},
		{"replacement", `func use(r Handle){*r.Borrow()=Counter{}}`},
		{"saved accessor", `func use(r Handle){f:=r.Borrow;_=f()}`},
		{"method expression", `func use(r Handle){_=Handle.Borrow(r)}`},
		{"bound message", `func use(r Handle){f:=r.Borrow().Count;_=f()}`},
		{"runtime method", `func use(r Handle){_=r.Borrow().Args()}`},
		{"init", `func use(r Handle){_=r.Borrow().Init()}`},
		{"run", `func use(r Handle){r.Borrow().Run(context.Background())}`},
		{"private method", `func(*Counter) private(){};func use(r Handle){r.Borrow().private()}`},
		{"promoted accessor", `type Wrapper struct{Handle};func use(r Wrapper){r.Borrow().Count()}`},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := checkSource(t, handleDefinitions+tc.source); got != "" {
				t.Fatal(got)
			}
		})
	}
}

// TestServiceHandleBoundary rejects invalid Ref targets and protected storage uses.
func TestServiceHandleBoundary(t *testing.T) {
	cases := []struct{ name, source string }{
		{"marked interface boxing", `func use(r Route)any{return r}`},
		{"marked interface conversion", `func use(r Route)any{return any(r)}`},
		{"nonservice", `func use(){var ref supervise.Ref[int];_=ref}`},
		{"unused nonservice", `var ref supervise.Ref[int]`},
		{"nonservice alias", `type Wrong = supervise.Ref[int]`},
		{"nonservice parameter", `func use(supervise.Ref[int]){}`},
		{"pointer target", `var ref supervise.Ref[*Counter]`},
		{"anonymous result", `type Bad struct{};func(Bad) Read()[]int{return nil}
func use(r supervise.Ref[struct{supervise.Service[supervise.NoArgs];Bad}])[]int{return r.Borrow().Read()}`},
		{"unused anonymous result", `type Bad struct{};func(Bad) Read()[]int{return nil}
var ref supervise.Ref[struct{supervise.Service[supervise.NoArgs];Bad}]`},
		{"protected field write", `func use(r *Route,h Handle){r.Worker=h}`},
		{"pointer field", `type Bad struct{im.Local;Worker *Handle}`},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := checkSource(t, handleDefinitions+tc.source); got == "" {
				t.Fatal("unsupported handle use accepted")
			}
		})
	}
}
