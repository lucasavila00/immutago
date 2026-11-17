package analyzer

import (
	"strings"
	"testing"
)

const frozenHandleFields = `
type Data struct { im.Frozen; N int }
func (d Data) Number() int { return d.N }
type Reader interface { im.FrozenInterface; Number() int }
type Text = im.FrozenString
type Holder struct {
 Text Text
 Texts [2]Text
 Values im.FrozenSlice[*Data]
 Index im.FrozenMap[string, *Data]
 Reader Reader
 Count int
}
type Local struct { im.Local; N int }
`

// TestFrozenHandleFields checks ordinary slots and their complete receiver and container forms.
func TestFrozenHandleFields(t *testing.T) {
	cases := map[string]string{
		"fields": `var saved Holder;func f(s Text,r Reader){saved=Holder{Text:s,Reader:r};saved.Text=s;
saved.Reader=nil;saved.Count++}`,
		"pointer receiver": `func(h *Holder) Set(s Text){h.Text=s;h.Count++}
func f(h Holder,s Text){h.Set(s);set:=h.Set;set(s);go h.Set(s);defer h.Set(s)}`,
		"value receiver": `func(h Holder) With(s Text)Holder{h.Text=s;h.Count++;return h}
func f(h Holder,s Text)Holder{return h.With(s)}`,
		"field address":       `func f(h *Holder,s Text){p:=&h.Text;*p=s;var q **Text=&p;**q=s;_=new(s)}`,
		"defined handle":      `type Other im.FrozenSlice[int];func f(value im.FrozenSlice[int])Other{return Other(value)}`,
		"embedded collection": `type Embedded struct{im.FrozenSlice[int]};func f(v Embedded)int{return v.Len()}`,
		"embedded string": `type Embedded struct{Text};func f(v Embedded)int{read:=v.Len;
return read()+Embedded.Len(v)}`,
		"embedded string pointer": `type Embedded struct{*Text};func f(v Embedded)int{return v.Len()}`,
		"nested arrays": `type Nested struct{Items [2]Holder};func f(s Text){v:=Nested{};v.Items[0].Text=s;copy:=v;
_=copy.Items[0].Text.Len()}`,
		"native containers": `func f(s Text){values:=[]Text{s};alias:=values;alias[0]=s;values=append(values,alias...)
entries:=map[Text]Holder{s:{Text:s}};entries[s]=Holder{Text:s};_=values[0];_=entries[s]}`,
		"channels": `func f(ch chan Text,s Text){ch<-s;value,ok:=<-ch;_=ok;select{case ch<-value:default:};
for item:=range ch{_=item.Len()}}`,
		"receive statements": `func f(ch chan Text){<-ch;select{case <-ch:default:};
select{case value,ok:=<-ch:_,_=value,ok;default:}}`,
		"interface channels": `func f(ch chan Reader,d Data){ch<-d;v:=<-ch;_=v.Number()}`,
		"tuples": `func pair()(Holder,Text){return Holder{},Text{}}
func f(h *Holder){other,text:=pair();h.Text,other.Text=text,h.Text;_,_=other,text}`,
		"generic holder": `type Box[T any]struct{Value T};func(b *Box[T])Set(v T){b.Value=v}
func f(s Text){b:=Box[Text]{Value:s};b.Set(s);_=b.Value.Len()}`,
		"generic containers": `func identity[T any](v T)T{return v};func f(h Holder){_=identity(h);_=identity(&h);
_=identity([]Text{h.Text})}`,
		"callbacks":      `func f(h Holder)func()Text{return func()Text{h.Count++;return h.Text}}`,
		"range bindings": `func f(values []Text){var i int;var text Text;for i,text=range values{};_,_=i,text}`,
		"service": `type Worker struct{supervise.Service[Holder];Name Text};
func(w *Worker)Run(ctx context.Context){w.Name=w.Args().Text;<-ctx.Done()}
func(w *Worker)Size()int{return w.Name.Len()}
func f(root *supervise.Supervisor,h Holder){ref,err:=supervise.Start[Worker](root,h);_=ref;
_=err}`,
	}
	for name, source := range cases {
		t.Run(name, func(t *testing.T) {
			imports := `import "context";import "github.com/lucasavila00/immutago/pkg/supervise";
var _ context.Context;var _ supervise.NoArgs;`
			if got := checkSource(t, imports+frozenHandleFields+source); got != "" {
				t.Fatal(got)
			}
		})
	}
}

// TestFrozenHandleFieldBoundaries keeps Local storage, target writes, and interface proofs separate.
func TestFrozenHandleFieldBoundaries(t *testing.T) {
	cases := map[string]struct{ source, want string }{
		"record field": {`type Bad struct{Value Data}`,
			"unmarked struct"},
		"local field": {`type Bad struct{Value [0]Local}`,
			"unmarked struct"},
		"projection field": {`type Bad struct{Value im.StringProjection}`,
			"unmarked struct"},
		"local generic": {`type Box[T any]struct{Value T};var _ Box[Local]`,
			"unmarked struct"},
		"frozen owner write": {`type Root struct{im.Frozen;Text Text};func f(p *Root,s Text){p.Text=s}`,
			"writes to immutable"},
		"frozen owner address": {`type Root struct{im.Frozen;Text Text};func f(p *Root){_= &p.Text}`,
			"interior addresses"},
		"collection target": {`func f(h Holder){h.Values.At(0).N++}`,
			"writes to immutable"},
		"builder alias": {`func f(s Text){values:=[]Text{s};alias:=values;_=alias;frozen:=im.FreezeSlice(values);
_=frozen}`,
			"copying"},
		"builder consumed": {`func f(s Text){values:=[]Text{s};frozen:=im.FreezeSlice(values);_=frozen;values[0]=s}`,
			"consumed"},
		"interface field": {`type Bad struct{*Data};func f(h *Holder){h.Reader=&Bad{}}`,
			"Frozen interface implementation"},
		"interface argument": {`type Bad struct{*Data};func take(Reader){};func f(){take(&Bad{})}`,
			"Frozen interface implementation"},
		"interface result": {`type Bad struct{*Data};func f()Reader{return &Bad{}}`,
			"Frozen interface implementation"},
		"interface tuple": {`type Bad struct{*Data};func pair()(*Bad,int){return &Bad{},1};
func f()(Reader,int){return pair()}`,
			"Frozen interface implementation"},
		"interface append": {`type Bad struct{*Data};func f(values []Reader){values=append(values,&Bad{})}`,
			"Frozen interface implementation"},
		"interface map": {`type Bad struct{*Data};func f(){_=map[string]Reader{"bad":&Bad{}}}`,
			"Frozen interface implementation"},
		"interface literal": {`type Bad struct{*Data};func f(){_=Holder{Reader:&Bad{}}}`,
			"Frozen interface implementation"},
		"interface select": {`type Bad struct{*Data};func f(ch chan Reader){select{case ch<-&Bad{}:}}`,
			"Frozen interface implementation"},
		"interface send": {`type Bad struct{*Data};func f(ch chan Reader){ch<-&Bad{}}`,
			"Frozen interface implementation"},
		"interface alias holder": {`type Bad struct{*Data};type Alias=Holder;func f(h *Alias){h.Reader=&Bad{}}`,
			"Frozen interface implementation"},
		"embedded projection": {`type Wrapped struct{im.Local;im.StringProjection};func f(v Wrapped)int{return v.Len()}`,
			"promoted immutable methods"},
		"interface assertion": {`func f(value any)Reader{return value.(Reader)}`,
			"Frozen interface implementation"},
	}
	for name, test := range cases {
		t.Run(name, func(t *testing.T) {
			if got := checkSource(t, frozenHandleFields+test.source); !strings.Contains(got, test.want) {
				t.Fatalf("got %q; want %q", got, test.want)
			}
		})
	}
}
