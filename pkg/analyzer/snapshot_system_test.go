package analyzer

import (
	"strings"
	"testing"
)

// TestSnapshotSystemForms checks the common payload and local read APIs together.
func TestSnapshotSystemForms(t *testing.T) {
	cases := []string{
		`type Data struct{im.Frozen; Name string; N int};var cell im.Cell[Data]
func load()im.Projection[*Data]{return cell.LoadPointer()}
func f(){cell.Store(Data{Name:"name",N:3});projection:=load()
reader:=NewDataProjection(projection);name:=reader.Name();_=name.Clone()
n:=reader.N();_=n;owned:=cell.Load().Unsafe();_=owned.Name}`,
		`var cell im.Cell[int];func f(){cell.Store(3);_=im.Value(cell.Load())}`,
		`type Data struct{im.Frozen; Name string};type Reply struct{im.Local; Name im.StringProjection; Score int}
func makeReply(v im.Projection[*Data])Reply{
return Reply{Name:NewDataProjection(v).Name(),Score:2}}`,
		`type Data struct{im.Frozen; Names [2]string};func f(){projection:=im.ProjectionOf(&Data{});i:=1
text:=NewDataProjection(projection).Names().At(i);_=text.Len()}`,
		`type Data struct{im.Frozen; Name string};func f(){projection:=im.ProjectionOf(&Data{})
if !im.IsNil(projection){_=NewDataProjection(projection).Name().Clone()}}`,
		`type Data struct{im.Frozen; Name string};type Pointer=*Data
func f(projection im.Projection[Pointer]){_=im.IsNil(projection);_=NewDataProjection(projection).Name().Len()}`,
		`func f(){frozen:=im.FreezeMap(map[string]int{"key":3});projection:=frozen.Projection()
value,ok:=projection.Get("key");_=ok;_=im.Value(value)
for key,value:=range projection.All(){_=im.StringOf(key).Clone();_=im.Value(value)}}`,
		`func f(){frozen:=im.FreezeSlice([]string{"text"});projection:=frozen.Projection()
_=im.StringOf(projection.At(0)).Clone()
for _,value:=range projection.All(){_=im.StringOf(value).Len()}}`,
		`func f(){frozen:=im.FreezeSlice([]int{1});projection:=frozen.Projection()
copy:=projection.Clone();copy[0]=2}`,
		`func f(){values:=im.Value(im.ProjectionOf([2]int{1,2}));_=values[0]}`,
		`type Data struct{im.Frozen; Text im.FrozenString};var cell im.Cell[Data]
func f(){text:="text";cell.Store(Data{Text:im.FreezeString(text)});_=text
projection:=NewDataProjection(cell.LoadPointer()).Text();_=projection.At(0);_=projection.Clone()}`,
		`func f(v im.SliceProjection[int]){values:=v.Clone();values=append(values,1)
frozen:=im.FreezeSlice(values);_=frozen}`,
		`func f(v im.MapProjection[int,int]){values:=v.Clone();values[1]=2;frozen:=im.FreezeMap(values);_=frozen}`,
		`func f(){text:=im.FreezeString("text");projection:=text.Projection()
_=text.EqualString("text");_=text.Equal(text);_=projection.Is("text");_=projection.Equal(projection)}`,
		`var saved string;func format(string){};func f(projection im.StringProjection,text im.FrozenString){
format(projection.Unsafe());format(text.Unsafe());saved=projection.Unsafe()}`,
		`type raw struct{values []int};type Data struct{im.Frozen; Raw im.UncheckedValue[raw]};var cell im.Cell[Data]
func f(){cell.Store(Data{Raw:im.Trust(&raw{values:[]int{1}})})
value:=cell.Load().Unsafe();_=im.Unwrap(value.Raw).values[0]}`,
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
		{`type Result struct{im.Local; Name string}`, "copied values require"},
		{`type Result struct{im.Local; Values im.FrozenSlice[int]}`, "copied values require"},
		{`type Result struct{im.Local; Text im.FrozenString}`, "copied values require"},
		{`func f(){_=im.Value(im.ProjectionOf(im.FreezeString("text")))}`,
			"copied values require"},
		{`type Result struct{im.Local; Text im.StringProjection};var cell im.Cell[Result]`,
			"Frozen storage cannot contain Local values"},
		{`type Result struct{im.Local; Text im.StringProjection};func f(){_=im.ProjectionOf(Result{})}`,
			"Frozen storage cannot contain Local values"},
		{`type Result struct{im.Local; N int};func f(){_=im.Projection[Result]{}.Unsafe()}`,
			"Frozen storage cannot contain Local values"},
		{`type Result struct{im.Local; Text im.StringProjection};type Data struct{im.Frozen; Result Result}`,
			"Frozen storage cannot contain Local values"},
		{`type Result struct{im.Local; Text im.StringProjection};var saved Result`, "package variables"},
		{`var values [0]im.Projection[string]`, "package variables"},
		{`var marker im.Local`, "package variables"},
		{`type Data struct{im.Local};type Pointer *Data`, "defined pointer types"},
		{`func f(){_=im.Value(im.ProjectionOf("text"))}`, "copied values require"},
		{`func f(){frozen:=im.FreezeMap(map[string]int{});projection:=frozen.Projection()
copy:=projection.Clone();_=copy}`,
			"copied values require"},
		{`func f(){frozen:=im.FreezeSlice([]int{1});projection:=frozen.Projection()
all:=projection.All();_=all}`,
			"direct range"},
		{`func f(){projection:=im.ProjectionOf("text");var boxed any=projection;_=boxed}`, "ordinary interfaces require"},
		{`func f(){projection:=im.ProjectionOf(1);go im.Value(projection)}`,
			"synchronous"},
	}
	for _, test := range cases {
		t.Run(test.source, func(t *testing.T) {
			if got := checkSource(t, test.source); !strings.Contains(got, test.want) {
				t.Fatalf("got %q; want %q", got, test.want)
			}
		})
	}
}

// TestProjectionUnsafe checks explicit shared access and the remaining target rules.
func TestProjectionUnsafe(t *testing.T) {
	const definitions = `type Data struct{im.Frozen;Name string;Names [2]string;Numbers [2]int;Next *Data;N int}
func identity(string)string{return ""};func read(*Data)int{return 0}
func f(projection im.Projection[*Data]){`
	for _, operation := range []string{
		`_ = projection.Unsafe()`,
		`_ = projection.Unsafe().Name`,
		`_ = projection.Unsafe().N`,
		`_ = im.ProjectionOf(projection.Unsafe().Next)`,
		`_ = identity(projection.Unsafe().Name)`,
		`_ = read(projection.Unsafe().Next)`,
	} {
		if got := checkSource(t, definitions+operation+`}`); got != "" {
			t.Fatal(got)
		}
	}
	for _, operation := range []string{
		`_ = projection.Unsafe().Numbers[:]`,
		`_ = &projection.Unsafe().N`,
		`projection.Unsafe().N = 1`,
		`projection.Unsafe().Numbers[0] = 1`,
		`raw := projection.Unsafe; _ = raw`,
		`_ = im.Projection[*Data].Unsafe(projection)`,
	} {
		if got := checkSource(t, definitions+operation+`}`); got == "" {
			t.Fatal("invalid shared access passed")
		}
	}
	for _, value := range []string{"1", `"text"`} {
		source := `func f(){p:=im.ProjectionOf(` + value + `);_=p.Unsafe()}`
		if got := checkSource(t, source); got != "" {
			t.Fatal(got)
		}
	}
}

// TestSnapshotServiceResults uses the same read proof as Local fields.
func TestSnapshotServiceResults(t *testing.T) {
	cases := []struct {
		source string
		reject bool
	}{
		{`type Data struct{im.Frozen;N int};func(*Counter) Data()Data{return Data{}}`, true},
		{`type Data struct{im.Frozen;Name string}
func(*Counter) Data()im.Projection[Data]{return im.ProjectionOf(Data{})}`, false},
		{`type Data struct{im.Frozen;Name string};func(*Counter) Data()Data{return Data{}}`, true},
		{`type Result struct{im.Local;Text im.StringProjection};func(*Counter) Data()Result{return Result{}}`, false},
		{`func(*Counter) Text()string{return "text"}`, true},
		{`func(*Counter) Local()[]im.Projection[int]{return nil}`, true},
	}
	for _, test := range cases {
		t.Run(test.source, func(t *testing.T) {
			if got := checkSource(t, serviceDefinitions+test.source); (got != "") != test.reject {
				t.Fatalf("got %q; reject %v", got, test.reject)
			}
		})
	}
}

// TestCellCallStatements applies the same statement forms to every payload type.
func TestCellCallStatements(t *testing.T) {
	for _, payload := range []string{"int", "string", "Data"} {
		for _, method := range []string{"Load", "Load().Unsafe"} {
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

// TestReferenceProjection applies the normal copied-value rules to generated reference getters.
func TestReferenceProjection(t *testing.T) {
	const record = `type Route struct{im.Frozen;Worker supervise.Ref[Counter];Count int}
func f(value im.Projection[*Route]){`
	for _, operation := range []string{
		`ref:=NewRouteProjection(value).Worker();_=ref`,
		`_ = NewRouteProjection(value).Count()`,
	} {
		if got := checkSource(t, serviceDefinitions+record+operation+`}`); got != "" {
			t.Fatal(got)
		}
	}
	for _, operation := range []string{
		`_ = value.Unsafe().Worker`,
		`_ = value.Unsafe().Count`,
		`saved = value.Unsafe().Worker`,
	} {
		source := serviceDefinitions + `var saved supervise.Ref[Counter];` + record + operation + `}`
		if got := checkSource(t, source); got != "" {
			t.Fatal(got)
		}
	}
}

// TestCopiedRecordBoundary applies one value rule to every consumer of copied data.
func TestCopiedRecordBoundary(t *testing.T) {
	const record = `type Count struct{im.Frozen;N int};type Alias=Count;`
	for _, source := range []string{
		`func f(){_=im.Value(im.ProjectionOf(Count{N:1}))}`,
		`func f(){_=im.Value(im.ProjectionOf([0]Alias{}))}`,
		`type Reply struct{im.Local;Count Count}`,
		`type Reply struct{im.Local;Counts [0]Alias}`,
		`func f(v im.SliceProjection[Count]){copy:=v.Clone();_=copy}`,
		`func f(v im.MapProjection[int,Count]){copy:=v.Clone();_=copy}`,
		`func f(v im.MapProjection[Count,int]){copy:=v.Clone();_=copy}`,
	} {
		if got := checkSource(t, record+source); !strings.Contains(got, "copied values require") {
			t.Fatal(got)
		}
	}
	for _, source := range []string{
		`type Reply struct{im.Local;N int};type Outer struct{im.Local;Value Reply}`,
		`var cell im.Cell[Count];func f(){cell.Store(Count{});_=NewCountProjection(cell.LoadPointer()).N()}`,
		`func f(v im.SliceProjection[[2]int]){_=v.Clone()}`,
		`func f(v im.MapProjection[[2]int,int]){_=v.Clone()}`,
	} {
		if got := checkSource(t, record+source); got != "" {
			t.Fatal(got)
		}
	}
}

// TestMarkersAreAnnotations keeps marker aliases usable only as embedded annotations in checked data.
func TestMarkersAreAnnotations(t *testing.T) {
	for _, source := range []string{
		`type Data struct{im.Frozen;Markers [2]im.Frozen}`,
		`type Data struct{im.Local;Markers [2]im.Local}`,
		`var cell im.Cell[im.Frozen]`,
		`func f(){_=im.ProjectionOf(im.Frozen{})}`,
		`func f(){_=im.FreezeSlice([]im.Frozen{{}})}`,
	} {
		if got := checkSource(t, source); !strings.Contains(got, "markers are annotations") {
			t.Fatal(got)
		}
	}
	if got := checkSource(t, `type FrozenMark=im.Frozen;type LocalMark=im.Local
 type Data struct{FrozenMark;N int};type Reply struct{LocalMark;N int}`); got != "" {
		t.Fatal(got)
	}
}
