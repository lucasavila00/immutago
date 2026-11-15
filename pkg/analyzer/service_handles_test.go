package analyzer

import "testing"

const handleDefinitions = serviceDefinitions + `
func(c *Counter) Count() int {return c.count}
type Handle = supervise.Ref[Counter]
type Route struct { im.Scoped; Worker Handle }
`

// TestServiceHandles checks handle copies without exposing the selected service pointer.
func TestServiceHandles(t *testing.T) {
	cases := []struct{ name, source string }{
		{"message", `func use(r Handle) int{return r.Current().Count()}`},
		{"parentheses", `func use(r Handle) int{return ((r.Current)()).Count()}`},
		{"ref result", `func(*Counter) Worker() Handle {return Handle{}}`},
		{"marked result", `func(*Counter) Route() Route {return Route{}}`},
		{"marked field", `func use(r Route) int{return r.Worker.Current().Count()}`},
		{"shared field", `type SharedRoute struct{im.Shared;Worker Handle}
func use(r *SharedRoute) int{return r.Worker.Current().Count()}`},
		{"native map", `func use(r Handle) int{m:=map[string]Handle{"a":r};v,ok:=m["a"];if !ok{return 0}
return v.Current().Count()}`},
		{"multiple copy", `func use(r Handle){a,b:=r,r;a,b=b,a;_,_=a,b}`},
		{"closure", `func use(r Handle){f:=func(){r.Current().Count()};go f();defer r.Current().Count()}`},
		{"channel", `func use(r Handle,ch chan Handle){ch<-r;other:=<-ch;other.Current().Count()}`},
		{"concrete callback", `func use(r Handle,f func(Handle)){f(r)}`},
		{"frozen field", `type Routes struct{im.Scoped;Workers im.FrozenSlice[Handle]}
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
		{"assertion", `func use(v any)Handle{return v.(Handle)}`},
		{"generic", `func get[T interface{Current()*Counter}](r T)*Counter{return r.Current()}
func use(r Handle){_=get(r)}`},
		{"type switch", `func use(v any){switch r:=v.(type){case Handle:r.Current().Count()}}`},
		{"nonservice", `func use(){var ref supervise.Ref[int];_=ref}`},
		{"generic wrapper", `type Box[T any] struct{Value T};func use(r Handle){_=Box[Handle]{r}}`},
		{"promoted accessor", `type Wrapper struct{Handle};func use(r Wrapper){r.Current().Count()}`},
		{"protected field write", `func use(r *Route,h Handle){r.Worker=h}`},
		{"pointer field", `type Bad struct{im.Scoped;Worker *Handle}`},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := checkSource(t, handleDefinitions+tc.source); got == "" {
				t.Fatal("unsupported handle use accepted")
			}
		})
	}
}
