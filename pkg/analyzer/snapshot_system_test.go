package analyzer

import (
	"strings"
	"testing"
)

// TestSnapshotSystemForms checks the common payload and local read APIs together.
func TestSnapshotSystemForms(t *testing.T) {
	cases := []string{
		`type Data struct{im.Frozen; Name string; N int};var cell im.Cell[Data]
func load()im.View[Data]{return cell.Load()}
func f(){cell.Store(Data{Name:"name",N:3});view:=load()
name:=im.Text(im.Select(view,func(d Data)string{return d.Name}));_=name.Clone()
n:=im.Copy(im.Select(view,func(d Data)int{return d.N}));_=n;owned:=cell.Shared();_=owned.Name}`,
		`var cell im.Cell[int];func f(){cell.Store(3);_=im.Copy(cell.Load())}`,
		`type Data struct{im.Frozen; Name string};type Reply struct{im.Local; Name im.StringView; Score int}
func makeReply(v im.View[Data])Reply{
return Reply{Name:im.Text(im.Select(v,func(d Data)string{return d.Name})),Score:2}}`,
		`type Data struct{im.Frozen; Names [2]string};func f(){view:=im.ViewOf(Data{});i:=1
text:=im.Text(im.Select(view,func(d Data)string{return d.Names[i]}));_=text.Len()}`,
		`type Data struct{im.Frozen; Name string};func f(){view:=im.ViewOf(&Data{})
if !im.IsNil(view){_=im.Text(im.Select(view,func(d *Data)string{return d.Name})).Clone()}}`,
		`func f(){frozen:=im.FreezeMap(map[string]int{"key":3});view:=im.Map(im.ViewOf(frozen))
value,ok:=view.Get("key");_=ok;_=im.Copy(value)
for key,value:=range view.All(){_=im.Text(key).Clone();_=im.Copy(value)}}`,
		`func f(){frozen:=im.FreezeSlice([]string{"text"});view:=im.Slice(im.ViewOf(frozen));_=im.Text(view.At(0)).Clone()
for _,value:=range view.All(){_=im.Text(value).Len()}}`,
		`func f(){frozen:=im.FreezeSlice([]int{1});view:=im.Slice(im.ViewOf(frozen));copy:=view.Clone();copy[0]=2}`,
		`type Data struct{im.Frozen; N int};func f(){value:=im.Copy(im.ViewOf(Data{N:3}));_=value.N}`,
		`type Data struct{im.Frozen; Text im.FrozenString};var cell im.Cell[Data]
func f(){text:="text";cell.Store(Data{Text:im.FreezeString(text)});_=text
view:=im.String(im.Select(cell.Load(),func(d Data)im.FrozenString{return d.Text}));_=view.At(0);_=view.Clone()}`,
		`func f(v im.SliceView[int]){values:=v.Clone();values=append(values,1);frozen:=im.FreezeSlice(values);_=frozen}`,
		`func f(v im.MapView[int,int]){values:=v.Clone();values[1]=2;frozen:=im.FreezeMap(values);_=frozen}`,
		`func f(){text:=im.FreezeString("text");view:=im.String(im.ViewOf(text))
_=text.EqualString("text");_=text.Equal(text);_=view.EqualString("text");_=view.Equal(view)}`,
		`var saved string;func format(string){};func f(view im.StringView,text im.FrozenString){
format(view.Unsafe());format(text.Unsafe());saved=view.Unsafe()}`,
		`type raw struct{values []int};type Data struct{im.Frozen; Raw im.UncheckedValue[raw]};var cell im.Cell[Data]
func f(){cell.Store(Data{Raw:im.Trust(&raw{values:[]int{1}})});value:=cell.Shared();_=im.Unwrap(value.Raw).values[0]}`,
	}
	for _, source := range cases {
		t.Run(source, func(t *testing.T) {
			if got := checkSource(t, source); got != "" {
				t.Fatal(got)
			}
		})
	}
}

// TestSnapshotCategories rejects the same invalid payload in each consuming API.
func TestSnapshotCategories(t *testing.T) {
	cases := []struct{ source, want string }{
		{`type Result struct{im.Local; Name string}`, "Copied values cannot contain shared data references"},
		{`type Result struct{im.Local; Values im.FrozenSlice[int]}`, "Copied values cannot contain shared data references"},
		{`type Result struct{im.Local; Text im.FrozenString}`, "Copied values cannot contain shared data references"},
		{`func f(){_=im.Copy(im.ViewOf(im.FreezeString("text")))}`, "Copied values cannot contain shared data references"},
		{`type Result struct{im.Local; Text im.StringView};var cell im.Cell[Result]`,
			"Frozen storage cannot contain Local values"},
		{`type Result struct{im.Local; Text im.StringView};func f(){_=im.ViewOf(Result{})}`,
			"Frozen storage cannot contain Local values"},
		{`type Result struct{im.Local; Text im.StringView};type Data struct{im.Frozen; Result Result}`,
			"Frozen storage cannot contain Local values"},
		{`type Result struct{im.Local; Text im.StringView};var saved Result`, "package variables"},
		{`var values [0]im.View[string]`, "package variables"},
		{`var marker im.Local`, "package variables"},
		{`func f(){_=im.Copy(im.ViewOf("text"))}`, "Copied values cannot contain shared data references"},
		{`func f(){frozen:=im.FreezeMap(map[string]int{});view:=im.Map(im.ViewOf(frozen));copy:=view.Clone();_=copy}`,
			"Copied values cannot contain shared data references"},
		{`func f(){frozen:=im.FreezeSlice([]int{1});view:=im.Slice(im.ViewOf(frozen));all:=view.All();_=all}`,
			"direct range"},
		{`func f(){view:=im.ViewOf("text");var boxed any=view;_=boxed}`, "boxing immutable storage"},
		{`type Data struct{im.Frozen;N int};func f(){view:=im.ViewOf(Data{});go im.Copy(view)}`, "synchronous"},
	}
	for _, test := range cases {
		t.Run(test.source, func(t *testing.T) {
			if got := checkSource(t, test.source); !strings.Contains(got, test.want) {
				t.Fatalf("got %q; want %q", got, test.want)
			}
		})
	}
}

// TestSelectRejectsOtherForms checks the rejecting path around each projection shape.
func TestSelectRejectsOtherForms(t *testing.T) {
	const definitions = `type Data struct{im.Frozen;Name string;Names [2]string};var index int
func f(){view:=im.ViewOf(Data{});`
	for _, operation := range []string{
		`_=im.Select(view,func(d Data)Data{return d})`,
		`_=im.Select(view,func(d Data)string{saved=d.Name;return d.Name})`,
		`_=im.Select(view,func(d Data)string{return d.Name[:]})`,
		`_=im.Select(view,func(d Data)string{return other(d.Name)})`,
		`_=im.Select(view,func(d Data)string{return d.Names[index]})`,
		`i:=0;_=im.Select(view,func(d Data)string{return d.Names[i+1]})`,
		`outside:=Data{};_=im.Select(view,func(d Data)string{return outside.Name})`,
		`project:=func(d Data)string{return d.Name};_=im.Select(view,project)`,
		`_=im.Select(view,func(d Data)(name string){return d.Name})`,
	} {
		source := definitions + operation + `};var saved string;func other(s string)string{return s}`
		t.Run(operation, func(t *testing.T) {
			if got := checkSource(t, source); !strings.Contains(got, "inline projection path") {
				t.Fatal(got)
			}
		})
	}
}

// TestSnapshotServiceResults uses the same read proof as Local fields.
func TestSnapshotServiceResults(t *testing.T) {
	cases := []struct {
		source string
		reject bool
	}{
		{`type Data struct{im.Frozen;N int};func(*Counter) Data()Data{return Data{}}`, false},
		{`type Data struct{im.Frozen;Name string};func(*Counter) Data()im.View[Data]{return im.ViewOf(Data{})}`, false},
		{`type Data struct{im.Frozen;Name string};func(*Counter) Data()Data{return Data{}}`, true},
		{`type Result struct{im.Local;Text im.StringView};func(*Counter) Data()Result{return Result{}}`, false},
		{`func(*Counter) Text()string{return "text"}`, true},
		{`func(*Counter) Local()[]im.View[int]{return nil}`, true},
	}
	for _, test := range cases {
		t.Run(test.source, func(t *testing.T) {
			if got := checkSource(t, serviceDefinitions+test.source); (got != "") != test.reject {
				t.Fatalf("got %q; reject %v", got, test.reject)
			}
		})
	}
}

// TestSelectSignatureRules keeps protected type syntax inside the normal source contract.
func TestSelectSignatureRules(t *testing.T) {
	cases := []string{
		`type Alias[T any]=[1]T;type S struct{im.Frozen;N int}
func f(v im.View[[1]S]){_=im.Select(v,func(s Alias[S])int{return s[0].N})}`,
		`func f(v im.View[[1]int]){_=im.Select(v,func(s [len([1]im.View[int]{})]int)int{return s[0]})}`,
		`type S struct{im.Frozen;N int}
func f(v im.View[[1]int],p *S){_=im.Select(v,func(s [len([1]*S{p})]int)int{return s[0]})}`,
	}
	for _, source := range cases {
		t.Run(source, func(t *testing.T) {
			if got := checkSource(t, source); got == "" {
				t.Fatal("unsupported Select signature passed")
			}
		})
	}
}

// TestCellCallStatements applies the same statement forms to every payload type.
func TestCellCallStatements(t *testing.T) {
	for _, payload := range []string{"int", "string", "Data"} {
		for _, method := range []string{"Load", "Shared"} {
			for _, statement := range []string{"go cell.%s()", "defer cell.%s()", "go use(cell.%s())", "defer use(cell.%s())"} {
				call := strings.ReplaceAll(statement, "%s", method)
				source := `type Data struct{im.Frozen;N int};var cell im.Cell[` + payload + `]
func use(any){};func f(){` + call + `}`
				t.Run(payload+"/"+call, func(t *testing.T) {
					if got := checkSource(t, source); !strings.Contains(got, "synchronous") {
						t.Fatalf("got %q; want synchronous call rejection", got)
					}
				})
			}
		}
	}
}

// TestReferenceProjection applies the same exact Select shape to service references.
func TestReferenceProjection(t *testing.T) {
	const record = `type Route struct{im.Frozen;Worker supervise.Ref[Counter];Count int}
func f(value im.View[Route]){`
	for _, operation := range []string{
		`ref:=im.Copy(im.Select(value,func(r Route)supervise.Ref[Counter]{return r.Worker}));_=ref`,
		`_=im.Copy(im.Select(value,func(r Route)int{return r.Count}))`,
	} {
		if got := checkSource(t, serviceDefinitions+record+operation+`}`); got != "" {
			t.Fatal(got)
		}
	}
	for _, operation := range []string{
		`_=im.Select(value,func(r Route)Route{return r})`,
		`_=im.Select(value,func(r Route)supervise.Ref[Counter]{saved=r.Worker;return r.Worker})`,
	} {
		source := serviceDefinitions + `var saved supervise.Ref[Counter];` + record + operation + `}`
		if got := checkSource(t, source); !strings.Contains(got, "inline projection path") {
			t.Fatal(got)
		}
	}
}
