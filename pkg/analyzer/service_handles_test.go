package analyzer

import "testing"

const handleDefinitions = serviceDefinitions + `
func(c *Counter) Count() int {return c.count}
type Handle = supervise.Ref[Counter]
type Route struct { im.Local; Worker Handle }
`

// TestServiceHandles checks handle copies without exposing the selected service pointer.
func TestServiceHandles(t *testing.T) {
	cases := []struct{ name, source string }{
		{"message", `func use(r Handle) int{return r.Current().Count()}`},
		{"parentheses", `func use(r Handle) int{return ((r.Current)()).Count()}`},
		{"ref result", `func(*Counter) Worker() Handle {return Handle{}}`},
		{"marked result", `func(*Counter) Route() Route {return Route{}}`},
		{"marked field", `func use(r Route) int{return r.Worker.Current().Count()}`},
		{"anonymous target", `type Reader struct{};func(Reader) Count()int{return 1}
func use(r supervise.Ref[struct{supervise.Service[supervise.NoArgs];Reader}])int{return r.Current().Count()}`},
		{"shared field", `type SnapshotRoute struct{im.Frozen;Worker Handle}
func use(r *SnapshotRoute) int{return r.Worker.Current().Count()}`},
		{"native map", `func use(r Handle) int{m:=map[string]Handle{"a":r};v,ok:=m["a"];if !ok{return 0}
return v.Current().Count()}`},
		{"multiple copy", `func use(r Handle){a,b:=r,r;a,b=b,a;_,_=a,b}`},
		{"closure", `func use(r Handle){f:=func(){r.Current().Count()};go f();defer r.Current().Count()}`},
		{"discarded message result", `func(*Counter) Worker()Handle{return Handle{}}
func use(r Handle){go r.Current().Worker();defer r.Current().Worker()}`},
		{"channel", `func use(r Handle,ch chan Handle){ch<-r;other:=<-ch;other.Current().Count()}`},
		{"concrete callback", `func use(r Handle,f func(Handle)){f(r)}`},
		{"concrete generic parameter", `func use[T any](r Handle,v T)Handle{return r}
func call(r Handle){use(r,1).Current().Count()}`},
		{"concrete generic field", `type Box[T any]struct{Worker Handle;Value T}
func use(r Handle){box:=Box[int]{Worker:r};box.Worker.Current().Count()}`},
		{"concrete generic alias", `type Alias[T any]=Handle;func use(r Alias[int]){r.Current().Count()}`},
		{"unrelated bodyless function", `func cpuID()uint64`},
		{"callback result", `func use(f func()Handle)int{next:=f;return next().Current().Count()}`},
		{"concrete iterator", `func use(r Handle){
for next:=range func(yield func(Handle)bool){yield(r)}{next.Current().Count()}}`},
		{"producer interface", `type Producer interface{Worker()Handle}
func pass(p Producer)Producer{return p}
func use(p Producer)int{next:=p;return pass(next).Worker().Current().Count()}`},
		{"native operations", `func use(r Handle){rs:=make([]Handle,1);rs=append(rs,r);copy(rs,rs);clear(rs)
m:=make(map[Handle]int);m[r]=1;delete(m,r);_=len(rs);_=cap(rs)}`},
		{"service reference state", `type Pool struct{supervise.Service[Handle];Workers map[string]Handle}
func(p *Pool) Update(r Handle){p.Workers["x"]=r;self:=p;f:=func(){self.Workers["y"]=r};f()
p.Args().Current().Count()}`},
		{"service callback", `func(c *Counter) Visit(f func(*Counter)){f(c)}
func use(r Handle){r.Current().Visit(func(c *Counter){c.count++})}`},
		{"frozen field", `type Routes struct{im.Frozen;Workers im.FrozenSlice[Handle]}
func use(r Routes)int{return r.Workers.At(0).Current().Count()}`},
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

// TestServiceHandleBoundary rejects indirect access and every escape of the service pointer.
func TestServiceHandleBoundary(t *testing.T) {
	cases := []struct{ name, source string }{
		{"pointer binding", `func use(r Handle){p:=r.Current();_=p}`},
		{"pointer return", `func use(r Handle)*Counter{return r.Current()}`},
		{"field read", `func use(r Handle)int{return r.Current().count}`},
		{"field write", `func use(r Handle){r.Current().count++}`},
		{"replacement", `func use(r Handle){*r.Current()=Counter{}}`},
		{"saved accessor", `func use(r Handle){f:=r.Current;_=f()}`},
		{"method expression", `func use(r Handle){_=Handle.Current(r)}`},
		{"bound message", `func use(r Handle){f:=r.Current().Count;_=f()}`},
		{"runtime method", `func use(r Handle){_=r.Current().Args()}`},
		{"init", `func use(r Handle){_=r.Current().Init()}`},
		{"run", `func use(r Handle){r.Current().Run(context.Background())}`},
		{"private method", `func(*Counter) private(){};func use(r Handle){r.Current().private()}`},
		{"interface", `func use(r Handle){var v interface{Current()*Counter}=r;_=v.Current()}`},
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
		{"conversion", `type Copy Handle;func use(r Handle){_=Copy(r)}`},
		{"conversion result", `type Copy Handle;func use(){_=Handle(Copy{})}`},
		{"nil conversion", `func use(){_=(*Handle)(nil)}`},
		{"nil container conversion", `func use(){_=[]Handle(nil)}`},
		{"assertion", `func use(v any)Handle{return v.(Handle)}`},
		{"callback boxing", `func use(r Handle){var v any=func()Handle{return r};_=v}`},
		{"callback assertion", `func use(v any){_=v.(func()Handle)}`},
		{"callback generic", `func keep[T any](v T)T{return v}
func use(r Handle){_=keep(func()Handle{return r})}`},
		{"iterator boxing", `func use(r Handle){var v any
for v=range func(yield func(Handle)bool){yield(r)}{_=v}}`},
		{"producer boxing", `type Producer interface{Worker()Handle};func use(p Producer){var v any=p;_=v}`},
		{"producer assertion", `type Producer interface{Worker()Handle};func use(v any){_=v.(Producer)}`},
		{"generic", `func get[T interface{Current()*Counter}](r T)*Counter{return r.Current()}
func use(r Handle){_=get(r)}`},
		{"type switch", `func use(v any){switch r:=v.(type){case Handle:r.Current().Count()}}`},
		{"nonservice", `func use(){var ref supervise.Ref[int];_=ref}`},
		{"unused nonservice", `var ref supervise.Ref[int]`},
		{"nonservice alias", `type Wrong = supervise.Ref[int]`},
		{"nonservice parameter", `func use(supervise.Ref[int]){}`},
		{"pointer target", `var ref supervise.Ref[*Counter]`},
		{"anonymous result", `type Bad struct{};func(Bad) Read()[]int{return nil}
func use(r supervise.Ref[struct{supervise.Service[supervise.NoArgs];Bad}])[]int{return r.Current().Read()}`},
		{"unused anonymous result", `type Bad struct{};func(Bad) Read()[]int{return nil}
var ref supervise.Ref[struct{supervise.Service[supervise.NoArgs];Bad}]`},
		{"bodyless parameter", `func leak(Handle)any`},
		{"bodyless result", `func leak()Handle`},
		{"bodyless callback", `func leak(func()Handle)any`},
		{"bodyless receiver", `type Wrapper struct{Worker Handle};func(Wrapper) leak()any`},
		{"generic wrapper", `type Box[T any] struct{Value T};func use(r Handle){_=Box[Handle]{r}}`},
		{"generic alias", `type Alias[T any]=Handle;var ref Alias[Handle]`},
		{"promoted accessor", `type Wrapper struct{Handle};func use(r Wrapper){r.Current().Count()}`},
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
